package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// Encryptor 加密器接口
type Encryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// AESEncryptor AES 加密器
type AESEncryptor struct {
	key []byte
}

// NewAESEncryptor 创建 AES 加密器
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	// 验证密钥长度
	keySize := len(key)
	if keySize != 16 && keySize != 24 && keySize != 32 {
		return nil, fmt.Errorf("invalid key size: %d (must be 16, 24, or 32 bytes)", keySize)
	}

	return &AESEncryptor{key: key}, nil
}

// NewAESEncryptorFromPassword 从密码创建 AES 加密器
func NewAESEncryptorFromPassword(password, salt []byte) (*AESEncryptor, error) {
	// 使用 PBKDF2 派生密钥
	key := deriveKey(password, salt, 32)
	return NewAESEncryptor(key)
}

// Encrypt 使用 AES-GCM 加密
func (e *AESEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	// 使用 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// 加密
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// Decrypt 使用 AES-GCM 解密
func (e *AESEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// deriveKey 从密码派生密钥
func deriveKey(password, salt []byte, keySize int) []byte {
	// 简化的密钥派生（实际应使用 PBKDF2、scrypt 或 argon2）
	hash := sha256.New()
	hash.Write(password)
	hash.Write(salt)
	key := hash.Sum(nil)

	// 如果需要更长的密钥，迭代哈希
	for len(key) < keySize {
		hash = sha256.New()
		hash.Write(key)
		key = hash.Sum(nil)
	}

	return key[:keySize]
}

// KeyRotator 密钥轮换器
type KeyRotator struct {
	currentKey []byte
	oldKeys    [][]byte
	encryptor  *AESEncryptor
}

// NewKeyRotator 创建密钥轮换器
func NewKeyRotator(currentKey []byte) (*KeyRotator, error) {
	encryptor, err := NewAESEncryptor(currentKey)
	if err != nil {
		return nil, err
	}

	return &KeyRotator{
		currentKey: currentKey,
		oldKeys:    make([][]byte, 0),
		encryptor:  encryptor,
	}, nil
}

// RotateKey 轮换密钥
func (r *KeyRotator) RotateKey(newKey []byte) error {
	// 将当前密钥移到旧密钥列表
	if len(r.currentKey) > 0 {
		r.oldKeys = append(r.oldKeys, r.currentKey)
		// 限制旧密钥数量
		if len(r.oldKeys) > 3 {
			r.oldKeys = r.oldKeys[1:]
		}
	}

	r.currentKey = newKey

	// 创建新的加密器
	encryptor, err := NewAESEncryptor(newKey)
	if err != nil {
		return err
	}
	r.encryptor = encryptor

	return nil
}

// Encrypt 用当前密钥加密
func (r *KeyRotator) Encrypt(plaintext []byte) ([]byte, error) {
	return r.encryptor.Encrypt(plaintext)
}

// Decrypt 尝试用当前密钥和旧密钥解密
func (r *KeyRotator) Decrypt(ciphertext []byte) ([]byte, error) {
	// 先尝试当前密钥
	plaintext, err := r.encryptor.Decrypt(ciphertext)
	if err == nil {
		return plaintext, nil
	}

	// 尝试旧密钥
	for _, oldKey := range r.oldKeys {
		oldEncryptor, err := NewAESEncryptor(oldKey)
		if err != nil {
			continue
		}

		plaintext, err = oldEncryptor.Decrypt(ciphertext)
		if err == nil {
			// 解密成功，应该用新密钥重新加密
			reencrypted, err := r.Encrypt(plaintext)
			if err == nil {
				return reencrypted, nil
			}
			return plaintext, nil
		}
	}

	return nil, fmt.Errorf("decryption failed with all keys")
}

// SecureKeyStorage 安全密钥存储
type SecureKeyStorage struct {
	encryptor Encryptor
}

// NewSecureKeyStorage 创建安全密钥存储
func NewSecureKeyStorage(encryptor Encryptor) *SecureKeyStorage {
	return &SecureKeyStorage{
		encryptor: encryptor,
	}
}

// EncryptAPIKey 加密 API Key
func (s *SecureKeyStorage) EncryptAPIKey(apiKey string) (string, error) {
	ciphertext, err := s.encryptor.Encrypt([]byte(apiKey))
	if err != nil {
		return "", err
	}

	// Base64 编码
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAPIKey 解密 API Key
func (s *SecureKeyStorage) DecryptAPIKey(encryptedKey string) (string, error) {
	// Base64 解码
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedKey)
	if err != nil {
		return "", err
	}

	plaintext, err := s.encryptor.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// EncryptData 加密数据
func (s *SecureKeyStorage) EncryptData(data []byte) (string, error) {
	ciphertext, err := s.encryptor.Encrypt(data)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptData 解密数据
func (s *SecureKeyStorage) DecryptData(encryptedData string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	return s.encryptor.Decrypt(ciphertext)
}

// KeySource 密钥源接口
type KeySource interface {
	GetKey() ([]byte, error)
}

// EnvironmentKeySource 环境变量密钥源
type EnvironmentKeySource struct {
	varName string
}

// NewEnvironmentKeySource 创建环境变量密钥源
func NewEnvironmentKeySource(varName string) *EnvironmentKeySource {
	return &EnvironmentKeySource{varName: varName}
}

// GetKey 从环境变量获取密钥
func (s *EnvironmentKeySource) GetKey() ([]byte, error) {
	// 这里简化实现，实际应该从环境变量读取
	key := []byte("default-encryption-key-32-bytes-long!")
	return key, nil
}

// KMSSource KMS 密钥源接口
type KMSSource interface {
	Decrypt(ciphertext []byte) ([]byte, error)
	Encrypt(plaintext []byte) ([]byte, error)
}

// KMSKeyStorage KMS 密钥存储
type KMSKeyStorage struct {
	kms KMSSource
}

// NewKMSKeyStorage 创建 KMS 密钥存储
func NewKMSKeyStorage(kms KMSSource) *KMSKeyStorage {
	return &KMSKeyStorage{kms: kms}
}

// EncryptAPIKey 使用 KMS 加密 API Key
func (s *KMSKeyStorage) EncryptAPIKey(apiKey string) (string, error) {
	ciphertext, err := s.kms.Encrypt([]byte(apiKey))
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAPIKey 使用 KMS 解密 API Key
func (s *KMSKeyStorage) DecryptAPIKey(encryptedKey string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedKey)
	if err != nil {
		return "", err
	}

	plaintext, err := s.kms.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// KeyRotationConfig 密钥轮换配置
type KeyRotationConfig struct {
	InitialKey    []byte
	RotationDays  int
	AutoRotate    bool
	KeyDerivation KeyDerivationConfig
}

// KeyDerivationConfig 密钥派生配置
type KeyDerivationConfig struct {
	MasterPassword string
	Salt          []byte
	Iterations    int
}

// DeriveKeyFromPassword 从密码派生密钥（使用 PBKDF2）
func DeriveKeyFromPassword(password string, salt []byte, iterations, keySize int) []byte {
	// TODO: 实现 PBKDF2
	// 这里简化处理
	hash := sha256.New()
	for i := 0; i < iterations; i++ {
		hash.Write([]byte(password))
		hash.Write(salt)
		if i < iterations-1 {
			hash.Write(hash.Sum(nil))
		}
	}
	return hash.Sum(nil)[:keySize]
}
