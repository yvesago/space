package models

import (
    "strings"

    "github.com/jinzhu/gorm"
    "golang.org/x/crypto/bcrypt"
)

const (
    PublicClient        string = "public"
    ConfidentialClient  string = "confidential"
)

type Client struct {
    Model
    UUID string                 `gorm:"not null;unique;index" validate:"omitempty,uuid4" json:"id"`
    Name string                 `gorm:"not null;unique;index" validate:"required,min=3,max=20" json:"name"`
    Description string          `json:"description"`
    Key string                  `gorm:"not null;unique;index" json:"-"`
    Secret string               `gorm:"not null" validate:"required" json:"-"`
    Scopes string               `gorm:"not null" validate:"required" json:"-"`
    CanonicalURI string         `gorm:"not null" validate:"required" json:"uri"`
    RedirectURI string          `gorm:"not null" validate:"required" json:"-"`
    Type string                 `gorm:"not null" validate:"required,client" json:"-"`
}

func validClientType(top interface{}, current interface{}, field interface{}, param string) bool {
    typeField := field.(string)
    if typeField != PublicClient && typeField != ConfidentialClient {
        return false
    }
    return true
}

func (client *Client) Authentic(secret string) bool {
    validSecret := bcrypt.CompareHashAndPassword([]byte(client.Secret), []byte(secret)) == nil
    return validSecret
}

func (client *Client) UpdateSecret(secret string) error {
    crypted, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
    if err == nil {
        client.Secret = string(crypted)
        return nil
    }
    return err
}

func (client *Client) BeforeSave(scope *gorm.Scope) error {
    return validateModel("validate", client)
}

func (client *Client) BeforeCreate(scope *gorm.Scope) error {
    scope.SetColumn("UUID", generateUUID())
    scope.SetColumn("Key", GenerateRandomString(32))
    if crypted, err := bcrypt.GenerateFromPassword([]byte(client.Secret), bcrypt.DefaultCost); err == nil {
        scope.SetColumn("Secret", crypted)
    } else {
        return err
    }
    return nil
}

func (client *Client) DefaultRedirectURI() string {
    return strings.Split(client.RedirectURI, "\n")[0]
}
