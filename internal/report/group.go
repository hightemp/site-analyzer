package report

import (
	"sort"
	"strings"

	"site-analyzer/internal/model"
)

const (
	cmsCategory   = "CMS"
	unknownCMS    = "Unknown"
	unknownCMSKey = "\x00unknown"
)

// GroupByCMS groups results by technologies in Wappalyzer's CMS category.
// Sites without a detected CMS are placed in the Unknown group.
func GroupByCMS(results []model.Result) []model.CMSGroup {
	groupsByKey := make(map[string]*model.CMSGroup)

	for _, result := range results {
		cmsNames := cmsNames(result.Technologies)
		if len(cmsNames) == 0 {
			cmsNames = []cmsName{{key: unknownCMSKey, display: unknownCMS}}
		}

		for _, cms := range cmsNames {
			group, ok := groupsByKey[cms.key]
			if !ok {
				group = &model.CMSGroup{CMS: cms.display, Sites: make([]model.Result, 0)}
				groupsByKey[cms.key] = group
			}
			group.Sites = append(group.Sites, result)
			group.SiteCount = len(group.Sites)
		}
	}

	groups := make([]model.CMSGroup, 0, len(groupsByKey))
	for _, group := range groupsByKey {
		groups = append(groups, *group)
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].CMS == unknownCMS {
			return false
		}
		if groups[j].CMS == unknownCMS {
			return true
		}
		return strings.ToLower(groups[i].CMS) < strings.ToLower(groups[j].CMS)
	})
	return groups
}

type cmsName struct {
	key     string
	display string
}

func cmsNames(technologies []model.Technology) []cmsName {
	seen := make(map[string]struct{})
	result := make([]cmsName, 0)
	for _, technology := range technologies {
		name := strings.TrimSpace(technology.Name)
		if name == "" || !hasCMSCategory(technology.Categories) {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, cmsName{key: key, display: name})
	}
	return result
}

func hasCMSCategory(categories []string) bool {
	for _, category := range categories {
		if strings.EqualFold(strings.TrimSpace(category), cmsCategory) {
			return true
		}
	}
	return false
}
