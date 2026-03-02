// 用户认证与管理逻辑
package logic

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// User 用户结构
type User struct {
	Username string `json:"username"`
	Password string `json:"password"` // bcrypt hash
	IsAdmin  bool   `json:"isAdmin"`
}

// AuthLogic 认证逻辑
type AuthLogic struct {
	mu       sync.RWMutex
	users    map[string]*User
	filePath string
}

// AuthSession 认证会话
type AuthSession struct {
	Username string
	IsAdmin  bool
	Token    string
}

var authInstance *AuthLogic
var authOnce sync.Once

// GetAuthLogic 获取认证逻辑单例
func GetAuthLogic(dataDir string) *AuthLogic {
	authOnce.Do(func() {
		authInstance = &AuthLogic{
			users:    make(map[string]*User),
			filePath: filepath.Join(dataDir, "users.json"),
		}
		authInstance.load()
	})
	return authInstance
}

// InitAdmin 初始化管理员账号（如果不存在）
func (a *AuthLogic) InitAdmin(username, password string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.users[username]; exists {
		return nil // 已存在，不覆盖
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %v", err)
	}

	a.users[username] = &User{
		Username: username,
		Password: string(hash),
		IsAdmin:  true,
	}
	return a.saveLocked()
}

// Authenticate 验证用户名密码，返回会话信息
func (a *AuthLogic) Authenticate(username, password string) (*AuthSession, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	user, exists := a.users[username]
	if !exists {
		return nil, fmt.Errorf("用户名或密码错误")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("用户名或密码错误")
	}

	token := generateToken()
	return &AuthSession{
		Username: username,
		IsAdmin:  user.IsAdmin,
		Token:    token,
	}, nil
}

// ChangePassword 修改密码
func (a *AuthLogic) ChangePassword(username, oldPassword, newPassword string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	user, exists := a.users[username]
	if !exists {
		return fmt.Errorf("用户不存在")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return fmt.Errorf("旧密码错误")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %v", err)
	}

	user.Password = string(hash)
	return a.saveLocked()
}

// ListUsers 列出所有用户（不含密码）
func (a *AuthLogic) ListUsers() []map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]map[string]interface{}, 0, len(a.users))
	for _, u := range a.users {
		result = append(result, map[string]interface{}{
			"username": u.Username,
			"isAdmin":  u.IsAdmin,
		})
	}
	return result
}

// CreateUser 创建用户（管理员操作）
func (a *AuthLogic) CreateUser(username, password string, isAdmin bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.users[username]; exists {
		return fmt.Errorf("用户已存在")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %v", err)
	}

	a.users[username] = &User{
		Username: username,
		Password: string(hash),
		IsAdmin:  isAdmin,
	}
	return a.saveLocked()
}

// DeleteUser 删除用户（管理员操作）
func (a *AuthLogic) DeleteUser(username, operatorUsername string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if username == operatorUsername {
		return fmt.Errorf("不能删除自己")
	}

	if _, exists := a.users[username]; !exists {
		return fmt.Errorf("用户不存在")
	}

	delete(a.users, username)
	return a.saveLocked()
}

// UpdateUser 更新用户（管理员操作）
func (a *AuthLogic) UpdateUser(username string, newPassword string, isAdmin *bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	user, exists := a.users[username]
	if !exists {
		return fmt.Errorf("用户不存在")
	}

	if newPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("密码加密失败: %v", err)
		}
		user.Password = string(hash)
	}

	if isAdmin != nil {
		user.IsAdmin = *isAdmin
	}

	return a.saveLocked()
}

// load 从文件加载用户数据
func (a *AuthLogic) load() {
	data, err := os.ReadFile(a.filePath)
	if err != nil {
		return // 文件不存在，使用空数据
	}

	var users []*User
	if err := json.Unmarshal(data, &users); err != nil {
		return
	}

	for _, u := range users {
		a.users[u.Username] = u
	}
}

// saveLocked 保存用户数据到文件（调用前需持有锁）
func (a *AuthLogic) saveLocked() error {
	users := make([]*User, 0, len(a.users))
	for _, u := range a.users {
		users = append(users, u)
	}

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化用户数据失败: %v", err)
	}

	// 确保目录存在
	dir := filepath.Dir(a.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	return os.WriteFile(a.filePath, data, 0600)
}

// generateToken 生成随机token
func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
