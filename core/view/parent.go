package view

import (
	"fmt"
	"os"
	"strings"
)

// buildSectionParentMap resolves @parent placeholders for a layout and its @extends chain.
func (e *Engine) buildSectionParentMap(name string, seen []string, bags map[string]*stackBag, once map[string]bool) (map[string]string, error) {
	raw, err := os.ReadFile(e.pathFor(name))
	if err != nil {
		return nil, err
	}
	content := string(raw)
	content, err = e.expandIncludes(content, seen, bags, once)
	if err != nil {
		return nil, err
	}
	content, err = e.expandComponents(content, seen, bags, once)
	if err != nil {
		return nil, err
	}
	content, err = e.expandEach(content, seen, bags, once)
	if err != nil {
		return nil, err
	}
	content = expandProps(content)
	content = applyOnce(content, once)
	content = collectStacks(content, bags)

	parentSources := extractParentSectionSources(content)

	if match := reExtends.FindStringSubmatch(content); len(match) == 2 {
		grandparent, err := e.buildSectionParentMap(match[1], seen, bags, once)
		if err != nil {
			return nil, err
		}
		for name, body := range extractSections(content) {
			parentSources[name] = strings.ReplaceAll(body, "@parent", grandparent[name])
		}
	} else {
		for name, body := range extractSections(content) {
			parentSources[name] = strings.ReplaceAll(body, "@parent", parentSources[name])
		}
	}

	return parentSources, nil
}

// extractParentSectionSources collects section bodies and @yield defaults from layout content.
func extractParentSectionSources(content string) map[string]string {
	sections := map[string]string{}

	for _, match := range reSectionShort.FindAllStringSubmatch(content, -1) {
		if len(match) == 3 {
			sections[match[1]] = match[2]
		}
	}
	for _, match := range reSectionShortVar.FindAllStringSubmatch(content, -1) {
		if len(match) == 3 {
			sections[match[1]] = "{{ $" + match[2] + " }}"
		}
	}
	for _, match := range reSectionShow.FindAllStringSubmatch(content, -1) {
		if len(match) == 3 {
			sections[match[1]] = strings.TrimSpace(match[2])
		}
	}
	for _, match := range reSection.FindAllStringSubmatch(content, -1) {
		if len(match) == 3 {
			sections[match[1]] = strings.TrimSpace(match[2])
		}
	}
	for _, match := range reYieldDefault.FindAllStringSubmatch(content, -1) {
		if len(match) == 3 {
			if _, ok := sections[match[1]]; !ok {
				sections[match[1]] = match[2]
			}
		}
	}

	return sections
}

func applyParentToSections(sections, parentSections map[string]string) map[string]string {
	out := make(map[string]string, len(sections))
	for name, body := range sections {
		out[name] = strings.ReplaceAll(body, "@parent", parentSections[name])
	}
	return out
}

// resolveRootLayout walks the @extends chain and returns the root layout without applying child sections.
func (e *Engine) resolveRootLayout(name string, seen []string, bags map[string]*stackBag, once map[string]bool) (string, error) {
	for _, item := range seen {
		if item == name {
			return "", fmt.Errorf("circular view dependency involving [%s]", name)
		}
	}
	seen = append(seen, name)

	raw, err := os.ReadFile(e.pathFor(name))
	if err != nil {
		return "", fmt.Errorf("view [%s] not found", name)
	}
	content := string(raw)
	content, err = e.expandIncludes(content, seen, bags, once)
	if err != nil {
		return "", err
	}
	content, err = e.expandComponents(content, seen, bags, once)
	if err != nil {
		return "", err
	}
	content, err = e.expandEach(content, seen, bags, once)
	if err != nil {
		return "", err
	}
	content = applyOnce(content, once)
	content = collectStacks(content, bags)
	content = expandProps(content)

	if match := reExtends.FindStringSubmatch(content); len(match) == 2 {
		return e.resolveRootLayout(match[1], seen, bags, once)
	}

	return content, nil
}
