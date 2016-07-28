package models

import (
    "time"

    "github.com/jinzhu/gorm"
    "gopkg.in/bluesuncorp/validator.v5"
)

const (
    AccessToken               string = "access_token"
    RefreshToken              string = "refresh_token"
    GrantToken                string = "grant_token"
    ActionToken               string = "action_token"

    PublicScope               string = "public"
    ReadScope                 string = "read"
    ReadWriteScope            string = "read_write"
)

type Session struct {
    Model
    UUID string                 `gorm:"not null;unique;index" validate:"omitempty,uuid4" json:"-"`
    User User                   `gorm:"not null" validate:"exists" json:"-"`
    UserID uint                 `gorm:"not null" json:"-"`
    Client Client               `gorm:"not null" validate:"exists" json:"-"`
    ClientID uint               `gorm:"not null" json:"-"`
    Moment int64                `gorm:"not null" json:"moment"`
    ExpiresIn int64             `gorm:"not null;default:0" json:"expires_in"`
    Ip string                   `gorm:"not null;index" validate:"required" json:"-"`
    UserAgent string            `gorm:"not null" validate:"required" json:"-"`
    Invalidated bool            `gorm:"not null;default:false"`
    Token string                `gorm:"not null;unique;index" validate:"omitempty,alphanum" json:"token"`
    TokenType string            `gorm:"not null;index" validate:"required,token" json:"token_type"`
    Scopes string               `gorm:"not null" validate:"required,scope" json:"-"`
}

func validScope(top interface{}, current interface{}, field interface{}, param string) bool {
    scope := field.(string)
    if scope != PublicScope && scope != ReadScope && scope != ReadWriteScope {
        return false
    }
    return true
}

func validTokenType(top interface{}, current interface{}, field interface{}, param string) bool {
    tokenType := field.(string)
    if tokenType != AccessToken && tokenType != RefreshToken &&
            tokenType != GrantToken && tokenType != ActionToken {
        return false
    }
    return true
}

func (session *Session) BeforeSave(scope *gorm.Scope) error {
    validate := validator.New("validate", validator.BakedInValidators)
    validate.AddFunction("scope", validScope)
    validate.AddFunction("token", validTokenType)
    err := validate.Struct(session)
    if err != nil {
        return err
    }
    return nil
}

func (session *Session) BeforeCreate(scope *gorm.Scope) error {
    scope.SetColumn("Token", randStringBytesMaskImprSrc(64))
    scope.SetColumn("UUID", generateUUID())
    scope.SetColumn("Moment", time.Now().UTC().Unix())
    return nil
}
