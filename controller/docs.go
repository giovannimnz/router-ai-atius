package controller

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// RoleAdminUser is the minimum role required to access /docs
const docsMinRole = common.RoleAdminUser

var docsHTML []byte

func init() {
	docsFile := os.Getenv("OPENAPI_SPEC_PATH")
	if docsFile == "" {
		docsFile = "/app/docs/openapi.json"
	}
	if data, err := os.ReadFile(docsFile); err == nil {
		docsHTML = data
	}
}

func DocsHandler(c *gin.Context) {
	session := sessions.Default(c)
	role := session.Get("role")
	if role == nil {
		role = 0
	}
	if role.(int) < docsMinRole {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, string(docsHTML))
}

func DocsJSONHandler(c *gin.Context) {
	session := sessions.Default(c)
	role := session.Get("role")
	if role == nil {
		role = 0
	}
	if role.(int) < docsMinRole {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	docsFile := os.Getenv("OPENAPI_SPEC_PATH")
	if docsFile == "" {
		docsFile = "/app/docs/openapi.json"
	}
	data, err := os.ReadFile(docsFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var spec map[string]interface{}
	if err := json.Unmarshal(data, &spec); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, spec)
}
