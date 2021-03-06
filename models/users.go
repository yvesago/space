package models

import (
    "fmt"
    "strings"

    "golang.org/x/crypto/bcrypt"
    "github.com/jinzhu/gorm"
    "github.com/pquerna/otp"
    "github.com/pquerna/otp/totp"

    "github.com/earaujoassis/space/security"
)

type User struct {
    Model
    UUID string                 `gorm:"not null;unique;index" validate:"omitempty,uuid4" json:"-"`
    PublicId string             `gorm:"not null;unique;index" json:"public_id"`
    Username string             `gorm:"not null;unique;index" validate:"required,alphanum,max=60" json:"-"`
    FirstName string            `gorm:"not null" validate:"required,min=3,max=20" essential:"required,min=3,max=20" json:"first_name"`
    LastName string             `gorm:"not null" validate:"required,min=3,max=20" essential:"required,min=3,max=20" json:"last_name"`
    Email string                `gorm:"not null;unique;index" validate:"required,email" essential:"required,email" json:"email"`
    Passphrase string           `gorm:"not null" validate:"required" essential:"required,min=10" json:"-"`
    Active bool                 `gorm:"not null;default:false" json:"active"`
    Admin bool                  `gorm:"not null;default:false" json:"-"`
    Client Client               `gorm:"not null" validate:"exists" json:"-"`
    ClientID uint               `gorm:"not null" json:"-"`
    Language Language           `gorm:"not null" validate:"exists" json:"-"`
    LanguageID uint             `gorm:"not null" json:"-"`
    TimezoneIdentifier string   `gorm:"not null;default:'GMT'" json:"timezone_identifier"`
    CodeSecret string           `gorm:"not null" validate:"required" json:"-"`
    RecoverSecret string        `gorm:"not null" validate:"required" json:"-"`
}

func (user *User) Authentic(password, passcode string) bool {
    validPassword := bcrypt.CompareHashAndPassword([]byte(user.Passphrase), []byte(password)) == nil
    var validPasscode bool
    if codeSecret, err := security.Decrypt(defaultKey(), user.CodeSecret); err != nil {
        return false
    } else {
        validPasscode = totp.Validate(passcode, string(codeSecret))
    }
    return validPasscode && validPassword
}

func (user *User) UpdatePassword(password string) error {
    crypted, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err == nil {
        user.Passphrase = string(crypted)
        return nil
    }
    return err
}

func (user *User) GenerateCodeSecret() *otp.Key {
    key, err := totp.Generate(totp.GenerateOpts{
        Issuer:      "QuatroLabs.com",
        AccountName: user.Username,
    })
    codeSecret := key.Secret()
    if cryptedCodeSecret, err := security.Encrypt(defaultKey(), []byte(codeSecret)); err == nil {
        user.CodeSecret = string(cryptedCodeSecret)
    } else {
        user.CodeSecret = codeSecret
    }
    if err != nil {
        return nil
    }
    return key
}

func (user *User) GenerateRecoverSecret() string {
    var secret string = strings.ToUpper(fmt.Sprintf("%s-%s-%s-%s",
        GenerateRandomString(4),
        GenerateRandomString(4),
        GenerateRandomString(4),
        GenerateRandomString(4),))
    user.RecoverSecret = secret
    return secret
}

func (user *User) BeforeSave(scope *gorm.Scope) error {
    return validateModel("validate", user)
}

func (user *User) BeforeCreate(scope *gorm.Scope) error {
    scope.SetColumn("UUID", generateUUID())
    scope.SetColumn("PublicId", GenerateRandomString(32))
    if cryptedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Passphrase), bcrypt.DefaultCost); err == nil {
        scope.SetColumn("Passphrase", cryptedPassword)
    } else {
        return err
    }
    if cryptedRecoverSecret, err := bcrypt.GenerateFromPassword([]byte(user.RecoverSecret), bcrypt.DefaultCost); err == nil {
        scope.SetColumn("RecoverSecret", cryptedRecoverSecret)
    } else {
        return err
    }
    return nil
}
