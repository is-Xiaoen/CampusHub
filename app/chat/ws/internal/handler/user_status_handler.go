package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"activity-platform/app/chat/ws/hub"
)

// GetUserStatusHandler 获取用户状态处理器
func GetUserStatusHandler(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 只允许 GET 请求
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 从查询参数获取 user_id
		userIDStr := r.URL.Query().Get("user_id")
		if userIDStr == "" {
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "缺少 user_id 参数",
				"data":    nil,
			})
			return
		}

		// 获取用户状态
		status, err := h.GetUserStatus(userIDStr)
		if err != nil {
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"code":    0,
				"message": "success",
				"data": map[string]interface{}{
					"is_online":       false,
					"last_seen":       0,
					"last_online_at":  0,
					"last_offline_at": 0,
				},
			})
			return
		}

		// 转换字符串为整数
		lastSeen, _ := strconv.ParseInt(status["last_seen"].(string), 10, 64)
		lastOnlineAt, _ := strconv.ParseInt(status["last_online_at"].(string), 10, 64)
		lastOfflineAt, _ := strconv.ParseInt(status["last_offline_at"].(string), 10, 64)

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"code":    0,
			"message": "success",
			"data": map[string]interface{}{
				"is_online":       status["is_online"],
				"last_seen":       lastSeen,
				"last_online_at":  lastOnlineAt,
				"last_offline_at": lastOfflineAt,
			},
		})
	}
}

func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
