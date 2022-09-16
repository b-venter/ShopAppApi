package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
)

//Struct type for context's Value being sent to Middleware #2
type CtxVerify struct {
	Sub string
	Iss string
	Tkn string
}

//Middleware 1: MiddleJWT
// CustomClaims contains the data we want from the token.
type Claims struct {
	Scope string   `json:"scope"`
	Iss   string   `json:"iss"`
	Sub   string   `json:"sub"`
	Aud   []string `json:"aud"`
	Iat   int64    `json:"iat"`
	Exp   int64    `json:"exp"`
	Azp   string   `json:"azp"`
}

// Validate satisfy validator.CustomClaims interface
func (c Claims) Validate(ctx context.Context) error {
	return nil
}

// middleJWT is a middleware that will check the validity of our JWT.
func middleJWT(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {

		c.Get("Authorization")

		issuerURL, err := url.Parse(os.Getenv("AUTH0_ISS"))
		if err != nil {
			fmt.Println("Middleware: issuer error: ", err)
			return echo.ErrInternalServerError
		}
		aud := os.Getenv("AUTH0_AUD")
		provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

		// c holds Response and Request
		jwtValidator, err := validator.New(
			provider.KeyFunc,
			validator.RS256,
			issuerURL.String(),
			[]string{aud},
			validator.WithCustomClaims(
				func() validator.CustomClaims {
					return &Claims{}
				},
			),
		)

		if err != nil {
			fmt.Println("Middleware: jwtValidator error: ", err)
			return echo.ErrInternalServerError
		}

		reqToken := c.Request().Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer ")

		//Test for token present
		if len(splitToken) < 2 {
			return echo.ErrUnauthorized
		}
		tkn := splitToken[1]

		vr, err := jwtValidator.ValidateToken(c.Request().Context(), tkn)

		if err != nil {
			fmt.Println("Middleware: jwtValidation error: ", err)
			return echo.ErrUnauthorized
		}

		if vr == nil {
			return echo.ErrUnauthorized //Not allowed to proceed. Might be nice to break it down further to "no token", etc
		}

		//Retrieve claims as a struct
		a := vr.(*validator.ValidatedClaims)

		//Ammend context to contain CtxVerify
		ver := CtxVerify{a.RegisteredClaims.Subject, a.RegisteredClaims.Issuer, tkn}
		con := c.Request()
		conctx := con.Context()
		c.SetRequest(con.WithContext(context.WithValue(conctx, "verify", ver)))

		return next(c) //Proceed to next.
	}
}

//Type for unpacking json
//map[email:"" email_verified: bool family_name:"" given_name:"" locale:"" name:"" nickname:"" picture:"" sub:"" updated_at:""]

type ujson map[string]interface{}

//Middleware 2: MiddleUser
//Matches sub with user's db,
func middleUser(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {

		//Get Sub from context, convert from interface to string
		ver := c.Request().Context().Value("verify").(CtxVerify)

		//SetCache returns false only if user not exist
		ok := ver.setCache()
		if !ok {
			return echo.ErrUnauthorized
		}

		//Set db for queries to that of the user in Context Value
		edb := cache[ver.Sub].db

		con := c.Request()
		conctx := con.Context()
		c.SetRequest(con.WithContext(context.WithValue(conctx, "db", edb)))

		return next(c) //Proceed to next.
	}
}

func middleAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		//Get Sub from context, convert from interface to string
		ver := c.Request().Context().Value("verify").(CtxVerify)

		//SetCache returns false only if user not exist
		ok := ver.setCache()
		if !ok {
			return echo.ErrUnauthorized
		}

		//Send sub. Each /admin query should verify role from cache
		con := c.Request()
		conctx := con.Context()
		c.SetRequest(con.WithContext(context.WithValue(conctx, "sub", ver.Sub)))

		return next(c) //Proceed to next.
	}
}

//This fnc sets the users in cache. Only returns false if subjct's email
// does not exist in the users db
func (cv CtxVerify) setCache() bool {

	sub := cv.Sub
	tkn := cv.Tkn
	iss := cv.Iss

	//Check if sub is a key which has data in cache
	usr := cache[sub]

	//Retrieve info from /userinfo if not present
	if usr.email == "" {
		//If not: query /userinfo and db
		u := getUser(tkn, iss)
		if sub != u["sub"] {
			return false //sub from token should match sub from Auth0's /userinfo
		}

		//Retrieve user info from db _system
		dbx, ctx := aranDB(ah, "_system")
		var execQ []d

		eml := u["email"]

		bind := d{"email": eml}
		q := "FOR d in users FILTER d.email == @email RETURN {db: d.db, email: d.email, role:d.role}"

		if ct {
			data := aranQuery{q, bind, dbx, ctx}
			execQ = data.aranQ()
		} else {
			fmt.Println("Failed to connect. Troubleshoot connection to ", ah)
			return false
		}

		em := fmt.Sprintf("%v", execQ[0]["email"])
		edb := fmt.Sprintf("%v", execQ[0]["db"])
		er := fmt.Sprintf("%v", execQ[0]["role"])

		if edb == "" {
			return false //user does not exist in system db
		}

		b := user{em, edb, er}
		cache[sub] = b

		return true

	}

	return true

}

//Retrieve /userinfo from Auth0 based on Bearer token
func getUser(tok string, iss string) ujson {
	url := iss + "userinfo"

	//curl --request GET --url 'https://YOUR_DOMAIN/userinfo' --header 'Authorization: Bearer {ACCESS_TOKEN}' --header 'Content-Type: application/json'
	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("authorization", "Bearer "+tok)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		fmt.Println(err)
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	//Get as json
	var bodyj ujson
	if err := json.Unmarshal(body, &bodyj); err != nil {
		fmt.Println("Error providing json: ", err)
	}

	return bodyj
}
