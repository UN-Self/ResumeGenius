package intake

import (
	"strconv"

	"github.com/UN-Self/ResumeGenius/backend/internal/shared/response"
	"github.com/gin-gonic/gin"
)

const CodeSSHKeyNotFound = 1007

type SSHHandler struct {
	sshSvc *SSHKeyService
}

func NewSSHHandler(sshSvc *SSHKeyService) *SSHHandler {
	return &SSHHandler{sshSvc: sshSvc}
}

type createSSHKeyReq struct {
	Alias      string `json:"alias" binding:"required"`
	PrivateKey string `json:"private_key" binding:"required"`
}

func (h *SSHHandler) CreateSSHKey(c *gin.Context) {
	var req createSSHKeyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, CodeParamInvalid, "alias and private_key are required")
		return
	}

	key, err := h.sshSvc.Create(userID(c), req.Alias, req.PrivateKey)
	if err != nil {
		response.Error(c, CodeInternalError, "failed to create SSH key")
		return
	}

	response.Success(c, key)
}

func (h *SSHHandler) ListSSHKeys(c *gin.Context) {
	keys, err := h.sshSvc.List(userID(c))
	if err != nil {
		response.Error(c, CodeInternalError, "failed to list SSH keys")
		return
	}

	response.Success(c, keys)
}

func (h *SSHHandler) DeleteSSHKey(c *gin.Context) {
	keyID, err := strconv.ParseUint(c.Param("key_id"), 10, 64)
	if err != nil {
		response.Error(c, CodeParamInvalid, "invalid key_id")
		return
	}

	if err := h.sshSvc.Delete(userID(c), uint(keyID)); err != nil {
		response.Error(c, CodeSSHKeyNotFound, "SSH key not found or in use")
		return
	}

	response.Success(c, nil)
}
