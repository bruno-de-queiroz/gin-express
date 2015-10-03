package express

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/contrib/sessions"
	"os"
	"path"
	"time"
	"github.com/olebedev/config"
	"net/mail"
	"github.com/gin-gonic/contrib/cors"
	"errors"
	"log"
)

type Environment int

const (
	DEVELOPMENT Environment = iota
	PRODUCTION
	TEST
)

var (
	environments = [3]string{"development", "production", "test"}
	modes = [3]string{"debug", "release", "test"}
)

func EnvironmentFromString(s string) Environment {
	for i, v := range environments {
		if v == s {
			return Environment(i)
		}
	}
	return Environment(0)
}

func (e Environment) String() string {
	return environments[e]
}

func (e Environment) Mode() string {
	return modes[e]
}

func debug(format string, values ...interface{}) {
	if gin.IsDebugging() {
		log.Printf("[APP-debug] " + format, values...)
	}
}

type GinExpress struct {
	*gin.Engine
	Base    string
	Config  *config.Config
	Session sessions.Session
	Logger  Logger
	Mailer  Mailer
}

func session(config *config.Config) (s sessions.Store, err error) {

	secret := []byte(config.UString("secret", "go-default-secret"))
	t := config.UString("type", "cookie")

	switch t {
	case "cookie":
		s = sessions.NewCookieStore(secret)
	case "redis":
		s, err = sessions.NewRedisStore(
			config.UInt("size"),
			config.UString("network"),
			config.UString("address"),
			config.UString("password"),
			secret,
		)
	default:
		err = errors.New("No session type allowed.")
	}

	if err != nil {
		return nil, err
	}

	o, _ := config.Get("options")

	if o != nil {
		s.Options(sessions.Options{
			Path:     o.UString("path"),
			Domain:   o.UString("domain"),
			MaxAge:   o.UInt("max_age", int(30 * time.Minute)),
			Secure:   o.UBool("secure"),
			HttpOnly: o.UBool("http_only"),
		})
	}

	return s, nil
}

func (a *GinExpress) Run() {
	a.Engine.Run(fmt.Sprintf("%s", a.Config.UString("config.host", ":9000")))
}

func interfaceToString(a []interface{}) (b[]string) {
	b = make([]string, 0)
	for _, v := range a {
		b = append(b, v.(string))
	}
	return b
}

func Express(p string) *GinExpress {

	env := EnvironmentFromString(os.Getenv("APPLICATION_ENV"))
	gin.SetMode(env.Mode())

	debug("Mode: %s", env.String())
	debug("Base path: %s", p)

	var (
		CONFIG_PATH = path.Join(p, "config")
		VIEWS_PATH = path.Join(p, "views")
		STATIC_PATH = path.Join(p, "public")
	)

	c, err := NewConfig(env, CONFIG_PATH)
	if err != nil {
		panic(err)
	}

	s := &GinExpress{}
	s.Engine = gin.New()
	s.Config = c

	s.Engine.Use(gin.Recovery())
	s.Engine.HTMLRender = NewHtmlMaster(VIEWS_PATH)

	lc, _ := c.Get("config.log")
	if lc != nil {

		s.Logger = NewLogger(&LogConfig{
			lc.UString("level"),
			lc.UString("file"),
			lc.UString("format"),
			lc.UBool("rotate", true),
		})

	} else {
		s.Logger = NewDefaultLogger()
	}

	s.Engine.Use(func(ctx *gin.Context) {
		start := time.Now()
		path := ctx.Request.URL.Path

		ctx.Next()

		end := time.Now()
		latency := end.Sub(start)

		clientIP := ctx.ClientIP()
		method := ctx.Request.Method
		statusCode := ctx.Writer.Status()
		comment := ctx.Errors.ByType(gin.ErrorTypePrivate).String()

		s.Logger.Info(statusCode,
			latency,
			clientIP,
			method,
			path,
			comment,
		)
	})

	cc, _ := c.Get("config.cors")
	if cc != nil {

		all := false
		origins := interfaceToString(cc.UList("allowed.origins"))
		if len(origins) == 0 || origins[0] == "*" {
			all = true
		}

		methods := interfaceToString(cc.UList("allowed.methods"))
		if len(methods) == 0 {
			methods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
		}

		headers := interfaceToString(cc.UList("allowed.headers"))
		exposed := interfaceToString(cc.UList("exposed"))

		s.Engine.Use(cors.New(cors.Config{
			AbortOnError: false,
			AllowAllOrigins: all,
			AllowedOrigins: origins,
			AllowedMethods: methods,
			AllowedHeaders: headers,
			ExposedHeaders: exposed,
			AllowCredentials: cc.UBool("credentials", false),
			MaxAge: time.Duration(cc.UInt("max_age", int(12 * time.Hour))),
		}))
	}

	sc, _ := c.Get("config.smtp")
	if sc != nil {

		s.Mailer = NewSimpleMailer(&MailerConfig{
			sc.UString("host"),
			sc.UInt("port"),
			sc.UString("username"),
			sc.UString("password"),
			&mail.Address{sc.UString("from.name"), sc.UString("from.email")},
			VIEWS_PATH,
		})
	}

	ss, _ := c.Get("config.session")
	if ss != nil {

		n := ss.UString("name", "go-default-session")

		if sess, err := session(ss); err != nil {

			debug("[ERR] Session", err.Error())

		} else {

			s.Engine.Use(sessions.Sessions(n, sess))
		}

	}


	s.Engine.Use(Gzip(DefaultCompression))

	s.Engine.Static("/public", STATIC_PATH)
	return s
}

