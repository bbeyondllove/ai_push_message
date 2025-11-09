package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"

	"ai_push_message/config"
	_ "ai_push_message/docs" // 导入 swagger 文档
	"ai_push_message/models"
	"ai_push_message/services"
	"ai_push_message/utils"
)

// PushUserHandler godoc
// @Summary 为指定用户通过HTTP推送已生成的推荐内容
// @Description 手动触发为指定用户通过HTTP推送已生成的推荐内容（不生成新的推荐内容）
// @Tags 推送
// @Accept json
// @Produce json
// @Param cid path string true "用户ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/push/user/{cid} [post]
func PushUserHandler(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	cid := chi.URLParam(r, "cid")
	if !utils.ValidateCID(w, cid) {
		return
	}

	// 获取用户的推荐内容
	recommendations, err := services.GetUserRecommendations(cid)
	if err != nil {
		utils.HandleServiceError(w, err, models.CodeNoRecommendData)
		return
	}

	// 检查推荐内容是否为空
	if len(recommendations) == 0 {
		utils.WriteErrorResponse(w, models.CodeNoRecommendData, map[string]interface{}{})
		return
	}

	// 执行推送
	if err := services.PushForCID(cfg, cid); err != nil {
		utils.WriteCustomErrorResponse(w, models.CodeServerError, err.Error(), map[string]interface{}{})
		return
	}

	// 返回成功响应
	utils.WriteSuccessResponse(w, map[string]interface{}{
		"cid":  cid,
		"tags": utils.FormatRecommendations(recommendations),
	})
}

// PushAllHandler godoc
// @Summary 为所有用户通过HTTP推送内容
// @Description 手动触发为所有用户通过HTTP推送已生成的推荐内容
// @Tags 推送
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/push/all [post]
func PushAllHandler(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	if err := services.PushAll(cfg); err != nil {
		utils.WriteCustomErrorResponse(w, models.CodeServerError, err.Error(), map[string]interface{}{})
		return
	}
	utils.WriteSuccessResponse(w, map[string]interface{}{})
}

// GenerateAllProfilesHandler godoc
// @Summary 生成所有用户的画像
// @Description 为所有用户生成画像
// @Tags 用户画像
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/profile/generate [post]
func GenerateAllProfilesHandler(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	if err := services.GenerateProfileForAllUsers(cfg); err != nil {
		utils.WriteCustomErrorResponse(w, models.CodeServerError, err.Error(), map[string]interface{}{})
		return
	}
	utils.WriteSuccessResponse(w, map[string]interface{}{
		"message": "Profile generation started for all users",
	})
}

// GenerateUserProfileHandler godoc
// @Summary 生成指定用户的画像
// @Description 为指定用户生成画像
// @Tags 用户画像
// @Accept json
// @Produce json
// @Param cid path string true "用户ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/profile/generate/{cid} [post]
func GenerateUserProfileHandler(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	cid := chi.URLParam(r, "cid")
	if !utils.ValidateCID(w, cid) {
		return
	}

	profile, _, err := services.GenerateProfileForUser(cfg, cid)
	if err != nil {
		utils.WriteCustomErrorResponse(w, models.CodeProfileGenError, err.Error(), map[string]interface{}{})
		return
	}

	// 如果用户没有画像（可能是因为没有数据），返回相应的消息
	if profile == nil {
		utils.WriteSuccessResponse(w, map[string]interface{}{
			"message":     "用户没有足够的数据来生成画像",
			"has_profile": false,
		})
		return
	}

	// 解析画像数据
	profileData, ok := utils.ParseProfileData(w, profile.ProfileRaw)
	if !ok {
		return
	}

	// 返回带有画像数据的响应
	utils.WriteSuccessResponse(w, profileData)
}

// GetUserProfileHandler godoc
// @Summary 获取用户画像
// @Description 获取指定用户的画像数据
// @Tags 用户画像
// @Accept json
// @Produce json
// @Param cid path string true "用户ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/profile/{cid} [get]
func GetUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	cid := chi.URLParam(r, "cid")
	if !utils.ValidateCID(w, cid) {
		return
	}

	// 获取用户画像
	profile, err := services.LoadUserProfile(cid)
	if err != nil {
		utils.WriteCustomErrorResponse(w, models.CodeServerError, err.Error(), map[string]interface{}{})
		return
	}

	if profile == nil {
		utils.WriteErrorResponse(w, models.CodeNoUserProfile, map[string]interface{}{})
		return
	}

	// 解析画像数据
	profileData, ok := utils.ParseProfileData(w, profile.ProfileRaw)
	if !ok {
		return
	}

	// 返回用户画像数据
	utils.WriteSuccessResponse(w, map[string]interface{}{
		"has_profile": true,
		"profile":     profileData,
	})
}

// GenerateUserRecommendationHandler godoc
// @Summary 为指定用户生成推荐内容（不推送）
// @Description 根据用户画像为指定用户生成推荐内容，但不推送给用户。会根据用户画像更新时间智能决定是否需要重新生成推荐内容
// @Tags 推荐内容
// @Accept json
// @Produce json
// @Param cid path string true "用户ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/recommendation/generate/{cid} [post]
func GenerateUserRecommendationHandler(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	cid := chi.URLParam(r, "cid")
	if !utils.ValidateCID(w, cid) {
		return
	}

	// 生成推荐内容
	recommendations, err := services.GenerateRecommendationsForUser(cfg, cid)
	if err != nil {
		utils.WriteCustomErrorResponse(w, models.CodeRecommendGenError, err.Error(), map[string]interface{}{})
		return
	}

	utils.WriteSuccessResponse(w, map[string]interface{}{
		"cid":    cid,
		"result": utils.FormatRecommendations(recommendations),
	})
}

// GenerateAllRecommendationsHandler godoc
// @Summary 为所有用户生成推荐内容
// @Description 根据用户画像为所有用户生成推荐内容
// @Tags 推荐内容
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/recommendation/generate [post]
func GenerateAllRecommendationsHandler(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	if err := services.GenerateRecommendationsForAllUsers(cfg); err != nil {
		utils.WriteCustomErrorResponse(w, models.CodeRecommendGenError, err.Error(), map[string]interface{}{})
		return
	}
	utils.WriteSuccessResponse(w, map[string]interface{}{
		"message": "已开始为所有用户生成推荐内容",
	})
}

// GetUserRecommendationHandler godoc
// @Summary 获取用户推荐内容
// @Description 获取指定用户的推荐内容
// @Tags 推荐内容
// @Accept json
// @Produce json
// @Param cid path string true "用户ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/recommendation/{cid} [get]
func GetUserRecommendationHandler(w http.ResponseWriter, r *http.Request) {
	cid := chi.URLParam(r, "cid")
	if !utils.ValidateCID(w, cid) {
		return
	}

	recommendations, err := services.GetUserRecommendations(cid)
	if err != nil {
		utils.HandleServiceError(w, err, models.CodeNoRecommendData)
		return
	}

	// 检查推荐内容是否为空
	if len(recommendations) == 0 {
		utils.WriteErrorResponse(w, models.CodeNoRecommendData, map[string]interface{}{})
		return
	}

	utils.WriteSuccessResponse(w, recommendations)
}

// RefreshUserRecommendationHandler godoc
// @Summary 强制刷新用户推荐内容
// @Description 强制重新生成用户的推荐内容，不检查画像更新时间，直接重新生成和更新用户画像和推荐内容
// @Tags 推荐内容
// @Accept json
// @Produce json
// @Param cid path string true "用户ID"
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /api/recommendation/refresh/{cid} [post]
func RefreshUserRecommendationHandler(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	cid := chi.URLParam(r, "cid")
	if !utils.ValidateCID(w, cid) {
		return
	}

	// 强制刷新用户推荐内容
	recommendations, err := services.RefreshUserRecommendations(cfg, cid)
	if err != nil {
		utils.WriteCustomErrorResponse(w, models.CodeRecommendGenError, err.Error(), map[string]interface{}{})
		return
	}

	// 检查推荐内容是否为空
	if len(recommendations) == 0 {
		utils.WriteErrorResponse(w, models.CodeNoRecommendData, map[string]interface{}{
			"cid": cid,
		})
		return
	}

	utils.WriteSuccessResponse(w, map[string]interface{}{
		"cid":     cid,
		"result":  utils.FormatRecommendations(recommendations),
		"message": "推荐内容已强制刷新",
	})
}

func RegisterRoutes(r *chi.Mux, cfg *config.Config) {
	// Swagger 文档
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // Swagger JSON 的 URL
	))

	r.Post("/api/push/user/{cid}", func(w http.ResponseWriter, r *http.Request) {
		PushUserHandler(w, r, cfg)
	})

	r.Post("/api/push/all", func(w http.ResponseWriter, r *http.Request) {
		PushAllHandler(w, r, cfg)
	})

	r.Post("/api/profile/generate", func(w http.ResponseWriter, r *http.Request) {
		GenerateAllProfilesHandler(w, r, cfg)
	})

	r.Post("/api/profile/generate/{cid}", func(w http.ResponseWriter, r *http.Request) {
		GenerateUserProfileHandler(w, r, cfg)
	})

	r.Get("/api/profile/{cid}", GetUserProfileHandler)

	r.Post("/api/recommendation/generate/{cid}", func(w http.ResponseWriter, r *http.Request) {
		GenerateUserRecommendationHandler(w, r, cfg)
	})

	r.Post("/api/recommendation/generate", func(w http.ResponseWriter, r *http.Request) {
		GenerateAllRecommendationsHandler(w, r, cfg)
	})

	r.Post("/api/recommendation/refresh/{cid}", func(w http.ResponseWriter, r *http.Request) {
		RefreshUserRecommendationHandler(w, r, cfg)
	})

	r.Get("/api/recommendation/{cid}", GetUserRecommendationHandler)
}
