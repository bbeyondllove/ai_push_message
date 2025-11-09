package services

import (
	"ai_push_message/models"
	"ai_push_message/repository"
)

func SearchGroupSummaries(keywords []string, lookbackDays, topN int) ([]models.RecommendationItem, error) {
	gs, err := repository.SearchGroupSummariesByKeywords(keywords, lookbackDays, topN)
	if err != nil {
		return nil, err
	}
	items := make([]models.RecommendationItem, 0, len(gs))
	for _, g := range gs {
		items = append(items, models.RecommendationItem{
			Source: "group_summary",
			Title:  g.GroupName + " | " + g.KeyTopics,
			URL:    "",
			Score:  0.8,
			RefID:  repository.IdToString(g.ID),
		})
	}
	return items, nil
}
