package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"backend/internal/account"
	"backend/internal/auth"
)

// Me reports the logged-in User's own Account.
func Me(accounts account.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		a, err := accounts.ByID(c.Request.Context(), auth.AccountID(c))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id":        a.ID,
			"name":      a.Identity.Name,
			"avatarUrl": a.Identity.AvatarURL,
		})
	}
}
