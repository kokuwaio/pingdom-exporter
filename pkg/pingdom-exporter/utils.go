package pingdom

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// TagByLabel represents a structured tag with a key, value, and format flag.
type TagByLabel struct {
	LabelKey   string
	LabelValue string
	Formatted  int
}

// ExtraLabel represents a mapping of custom labels with their corresponding values.
type ExtraLabel map[string]string

// TagLabel extracts a label key and value from a given string using a regex pattern.
// It returns a TagByLabel struct containing the extracted key, value, and a formatting flag.
func TagLabel(n string, f string) (TagByLabel, error) {
	regex, err := regexp.Compile(f)
	tl := TagByLabel{LabelKey: "", LabelValue: "", Formatted: 0}

	if err != nil {
		return tl, err
	}

	matches := regex.FindAllStringSubmatch(n, -1)

	if len(matches) > 0 {
		tl.LabelKey = matches[0][1]
		tl.LabelValue = matches[0][2]
		tl.Formatted = 1
	}
	return tl, nil
}

// toSnakeCase converts a string to snake_case format.
func toSnakeCase(input string) string {
	var sb strings.Builder

	// Iterate over each character in the input string.
	for i, r := range input {
		// Check if the character is an uppercase letter.
		if unicode.IsUpper(r) {
			// Add an underscore before uppercase letters (except at the beginning) and convert to lowercase.
			if i > 0 && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
				sb.WriteRune('_')
			}
			r = unicode.ToLower(r)
		}
		sb.WriteRune(r)
	}

	// Clean non-alphanumeric characters and consecutive underscores.
	result := sb.String()
	result = regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(result, "_")
	result = regexp.MustCompile(`_{2,}`).ReplaceAllString(result, "_")

	// Trim leading/trailing underscores.
	return strings.Trim(result, "_")
}

// ProcessExtraLabels converts a comma-separated string of labels into a map where
// the key is the original label and the value is its snake_case version.
// It also returns the list of keys in the order they were processed.
func ProcessExtraLabels(labels string) (ExtraLabel, []string) {
	extraLabels := make(ExtraLabel)
	var labelOrder []string

	// Split the input string by commas and process each label.
	for _, label := range strings.Split(labels, ",") {
		trimmedLabel := strings.TrimSpace(label)
		if trimmedLabel != "" {
			// Store the label in snake_case format
			snakeLabel := fmt.Sprintf("label_%s", toSnakeCase(trimmedLabel))
			extraLabels[trimmedLabel] = snakeLabel
			labelOrder = append(labelOrder, trimmedLabel) // Maintain the order of labels
		}
	}
	return extraLabels, labelOrder
}

// GetLabelNamesFromExtraLabels returns a slice of label names in snake_case format.
func GetLabelNamesFromExtraLabels(extraLabels ExtraLabel) []string {
	labelNames := make([]string, 0, len(extraLabels))
	for _, labelValue := range extraLabels {
		labelNames = append(labelNames, labelValue)
	}
	return labelNames
}

// GetExtraLabelsValues extracts values from resource labels based on extra labels.
// If a label is missing, an empty string is added instead. It takes the label order as input.
func GetExtraLabelsValues(resourceLabels map[string]string, extraLabels ExtraLabel, labelOrder []string) []string {
	var labelValues []string
	for _, key := range labelOrder { // Use the ordered list of keys
		if value, exists := resourceLabels[key]; exists {
			labelValues = append(labelValues, value)
		} else {
			labelValues = append(labelValues, "") // Add empty if label is missing
		}
	}
	return labelValues
}
