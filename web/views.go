package web

import (
    "fmt"
    "net/http"
    "net/url"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/gin-gonic/contrib/renders/multitemplate"
    "github.com/gin-gonic/contrib/sessions"

    "github.com/earaujoassis/space/config"
    "github.com/earaujoassis/space/models"
    "github.com/earaujoassis/space/oauth"
    "github.com/earaujoassis/space/services"
    "github.com/earaujoassis/space/feature"
    "github.com/earaujoassis/space/utils"
)

const (
    errorURI string = "%s?error=%s&state=%s"
)

var spaceCDN string

func createCustomRender() multitemplate.Render {
    render := multitemplate.New()
    render.AddFromFiles("satellite", "web/templates/default.html", "web/templates/satellite.html")
    render.AddFromFiles("error", "web/templates/default.html", "web/templates/error.html")
    return render
}

func ExposeRoutes(router *gin.Engine) {
    router.LoadHTMLGlob("web/templates/*.html")
    router.HTMLRender = createCustomRender()
    if config.IsEnvironment("production") && config.GetConfig("SPACE_CDN") != "" {
        spaceCDN = config.GetConfig("SPACE_CDN")
    } else {
        spaceCDN = "/public"
        router.Static("/public", "web/public")
    }
    store := sessions.NewCookieStore([]byte(config.GetConfig("SPACE_SESSION_SECRET")))
    store.Options(sessions.Options{
        Secure: config.IsEnvironment("production"),
        HttpOnly: true,
    })
    router.Use(sessions.Sessions("jupiter.session", store))
    views := router.Group("/")
    {
        views.GET("/", jupiterHandler)
        views.GET("/profile", jupiterHandler)

        views.GET("/signup", func(c *gin.Context) {
            c.HTML(http.StatusOK, "satellite", utils.H{
                "AssetsEndpoint": spaceCDN,
                "Title": " - Sign up",
                "Satellite": "io",
                "Data": utils.H{
                    "feature.gates": utils.H{
                        "user.create": feature.Active("user.create"),
                    },
                },
            })
        })

        views.GET("/signin", func(c *gin.Context) {
            c.HTML(http.StatusOK, "satellite", utils.H{
                "AssetsEndpoint": spaceCDN,
                "Title": " - Sign in",
                "Satellite": "ganymede",
            })
        })

        views.GET("/signout", func(c *gin.Context) {
            session := sessions.Default(c)

            userPublicId := session.Get("userPublicId")
            if userPublicId != nil {
                session.Delete("userPublicId")
                session.Save()
            }

            c.Redirect(http.StatusFound, "/signin")
        })

        views.GET("/session", func(c *gin.Context) {
            session := sessions.Default(c)

            userPublicId := session.Get("userPublicId")
            if userPublicId != nil {
                c.Redirect(http.StatusFound, "/")
                return
            }

            var nextPath string = "/"
            var scope string = c.Query("scope")
            var grantType string = c.Query("grant_type")
            var code string = c.Query("code")
            var clientId string = c.Query("client_id")
            var _nextPath string = c.Query("_")
            //var state string = c.Query("state")

            if scope == "" || grantType == "" || code == "" || clientId == "" {
                // Original response:
                // c.String(http.StatusMethodNotAllowed, "Missing required parameters")
                c.Redirect(http.StatusFound, "/signin")
                return
            }
            if _nextPath != "" {
                if _nextPath, err := url.QueryUnescape(_nextPath); err == nil {
                    nextPath = _nextPath
                }
            }

            client := services.FindOrCreateClient("Jupiter")
            if client.Key == clientId && grantType == oauth.AuthorizationCode && scope == models.PublicScope {
                grantToken := services.FindSessionByToken(code, models.GrantToken)
                if grantToken.ID != 0 {
                    session.Set("userPublicId", grantToken.User.PublicId)
                    session.Save()
                    services.InvalidateSession(grantToken)
                    c.Redirect(http.StatusFound, nextPath)
                    return
                }
            }

            c.Redirect(http.StatusFound, "/signin")
        })

        views.GET("/authorize", authorizeHandler)
        views.POST("/authorize", authorizeHandler)

        views.GET("/error", func(c *gin.Context) {
            errorReason := c.Query("response_type")

            c.HTML(http.StatusOK, "error", utils.H{
                "AssetsEndpoint": spaceCDN,
                "errorReason": errorReason,
            })
        })

        views.POST("/token", func(c *gin.Context) {
            var grantType string = c.PostForm("grant_type")

            authorizationBasic := strings.Replace(c.Request.Header.Get("Authorization"), "Basic ", "", 1)
            client := oauth.ClientAuthentication(authorizationBasic)
            if client.ID == 0 {
                c.Header("WWW-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", c.Request.RequestURI))
                c.JSON(http.StatusUnauthorized, utils.H{
                    "error": oauth.AccessDenied,
                })
                return
            }

            switch grantType {
            // Authorization Code Grant
            case oauth.AuthorizationCode:
                result, err := oauth.AccessTokenRequest(utils.H{
                    "grant_type": grantType,
                    "code": c.PostForm("code"),
                    "redirect_uri": c.PostForm("redirect_uri"),
                    "client": client,
                })
                if err != nil {
                    c.JSON(http.StatusMethodNotAllowed, utils.H{
                        "error": result["error"],
                    })
                    return
                } else {
                    c.JSON(http.StatusOK, utils.H{
                        "user_id": result["user_id"],
                        "access_token": result["access_token"],
                        "token_type": result["token_type"],
                        "expires_in": result["expires_in"],
                        "refresh_token": result["refresh_token"],
                        "scope": result["scope"],
                    })
                    return
                }
                return
            // Refreshing an Access Token
            case oauth.RefreshToken:
                result, err := oauth.RefreshTokenRequest(utils.H{
                    "grant_type": grantType,
                    "refresh_token": c.PostForm("refresh_token"),
                    "scope": c.PostForm("scope"),
                    "client": client,
                })
                if err != nil {
                    c.JSON(http.StatusMethodNotAllowed, utils.H{
                        "error": result["error"],
                    })
                    return
                } else {
                    c.JSON(http.StatusOK, utils.H{
                        "user_id": result["user_id"],
                        "access_token": result["access_token"],
                        "token_type": result["token_type"],
                        "expires_in": result["expires_in"],
                        "refresh_token": result["refresh_token"],
                        "scope": result["scope"],
                    })
                    return
                }
                return
            // Resource Owner Password Credentials Grant
            // Client Credentials Grant
            case oauth.Password, oauth.ClientCredentials:
                c.JSON(http.StatusMethodNotAllowed, utils.H{
                    "error": oauth.UnsupportedGrantType,
                })
                return
            default:
                c.JSON(http.StatusBadRequest, utils.H{
                    "error": oauth.InvalidRequest,
                })
                return
            }
        })
    }
}

func jupiterHandler(c *gin.Context) {
    session := sessions.Default(c)
    userPublicId := session.Get("userPublicId")
    if userPublicId == nil {
        c.Redirect(http.StatusFound, "/signin")
        return
    }
    client := services.FindOrCreateClient("Jupiter")
    user := services.FindUserByPublicId(userPublicId.(string))
    actionToken := services.CreateAction(user, client,
        c.Request.RemoteAddr,
        c.Request.UserAgent(),
        models.ReadWriteScope)
    c.HTML(http.StatusOK, "satellite", utils.H{
        "AssetsEndpoint": spaceCDN,
        "Title": " - Mission control",
        "Satellite": "europa",
        "Data": utils.H {
            "action_token": actionToken.Token,
            "user_id": user.UUID,
        },
    })
}

func authorizeHandler(c *gin.Context) {
    var location string
    var responseType string
    var clientId string
    var redirectURI string
    var scope string
    var state string

    session := sessions.Default(c)
    userPublicId := session.Get("userPublicId")
    nextPath := url.QueryEscape(fmt.Sprintf("%s?%s", c.Request.URL.Path, c.Request.URL.RawQuery))
    if userPublicId == nil {
        location = fmt.Sprintf("/signin?_=%s", nextPath)
        c.Redirect(http.StatusFound, location)
        return
    }
    user := services.FindUserByPublicId(userPublicId.(string))
    if user.ID == 0 {
        session.Delete("userPublicId")
        session.Save()
        location = fmt.Sprintf("/signin?_=%s", nextPath)
        c.Redirect(http.StatusFound, location)
        return
    }

    responseType = c.Query("response_type")
    clientId = c.Query("client_id")
    redirectURI = c.Query("redirect_uri")
    scope = c.Query("scope")
    state = c.Query("state")

    if redirectURI == "" {
        redirectURI = "/error"
    }

    client := services.FindClientByKey(clientId)
    if client.ID == 0 {
        redirectURI = "/error"
        location = fmt.Sprintf("%s?error=%s&state=%s",
            redirectURI, oauth.UnauthorizedClient, state)
        c.Redirect(http.StatusFound, location)
        return
    }

    if scope != models.PublicScope && scope != models.ReadScope && scope != models.ReadWriteScope {
        scope = "public"
    }

    switch responseType {
    // Authorization Code Grant
    case oauth.Code:
        activeSessions := services.ActiveSessionsForClient(client.ID, user.ID)
        if c.Request.Method == "GET" && activeSessions == 0 {
            c.HTML(http.StatusOK, "satellite", utils.H{
                "AssetsEndpoint": spaceCDN,
                "Title": " - Authorize",
                "Satellite": "callisto",
                "Data": utils.H{
                    "first_name": user.FirstName,
                    "last_name": user.LastName,
                    "client_name": client.Name,
                    "client_uri": client.CanonicalURI,
                    "requested_scope": scope,
                },
            })
            return
        } else if c.Request.Method == "POST" || (activeSessions > 0 && c.Request.Method == "GET") {
            if c.PostForm("access_denied") == "true" {
                location = fmt.Sprintf(errorURI, redirectURI, oauth.AccessDenied, state)
                c.Redirect(http.StatusFound, location)
                return
            }
            result, err := oauth.AuthorizationCodeGrant(utils.H{
                "response_type": responseType,
                "client": client,
                "user": user,
                "ip": c.Request.RemoteAddr,
                "userAgent": c.Request.UserAgent(),
                "redirect_uri": redirectURI,
                "scope": scope,
                "state": state,
            })
            if err != nil {
                location = fmt.Sprintf(errorURI, redirectURI, result["error"], result["state"])
                c.Redirect(http.StatusFound, location)
            } else {
                location = fmt.Sprintf("%s?code=%s&scope=%s&state=%s",
                    redirectURI, result["code"], result["scope"], result["state"])
                c.Redirect(http.StatusFound, location)
            }
        } else {
            c.String(http.StatusNotFound, "404 Not Found")
        }
    // Implicit Grant
    case oauth.Token:
        location = fmt.Sprintf(errorURI,
            redirectURI, oauth.UnsupportedResponseType, state)
        c.Redirect(http.StatusFound, location)
        return
    default:
        location = fmt.Sprintf(errorURI,
            redirectURI, oauth.InvalidRequest, state)
        c.Redirect(http.StatusFound, location)
        return
    }
}
