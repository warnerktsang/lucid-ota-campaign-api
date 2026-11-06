package auth

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"hash"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	routing "github.com/go-ozzo/ozzo-routing/v2"
	"github.com/go-ozzo/ozzo-routing/v2/content"
	"github.com/stretchr/testify/assert"
	"github.com/warnerktsang/lucid-ota-campaign-api/internal/errors"
	"github.com/warnerktsang/lucid-ota-campaign-api/pkg/log"
)

const testSigningKey = "test-signing-key-0123456789abcdef"

// Regression tests for SEC-2041: the JWT validation boundary that previously relied on
// github.com/dgrijalva/jwt-go. Test tokens are built with the standard library only, so the
// expectations do not depend on the API of the replacement JWT library.

func b64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func encodeSegment(t *testing.T, v interface{}) string {
	b, err := json.Marshal(v)
	assert.NoError(t, err)
	return b64url(b)
}

// hmacToken builds a JWS compact token signed with the given HMAC hash and key.
func hmacToken(t *testing.T, header, claims map[string]interface{}, key string, h func() hash.Hash) string {
	signingInput := encodeSegment(t, header) + "." + encodeSegment(t, claims)
	mac := hmac.New(h, []byte(key))
	mac.Write([]byte(signingInput))
	return signingInput + "." + b64url(mac.Sum(nil))
}

// hs256Token builds an HS256 token for the given claims, the format issued by the service.
func hs256Token(t *testing.T, claims map[string]interface{}, key string) string {
	return hmacToken(t, map[string]interface{}{"alg": "HS256", "typ": "JWT"}, claims, key, sha256.New)
}

func validClaims() map[string]interface{} {
	return map[string]interface{}{
		"id":   "100",
		"name": "demo",
		"exp":  time.Now().Add(time.Hour).Unix(),
	}
}

// newTestRouter returns a router protected by Handler whose single route echoes the authenticated user.
func newTestRouter(key string) *routing.Router {
	logger, _ := log.NewForTest()
	router := routing.New()
	router.Use(errors.Handler(logger), content.TypeNegotiator(content.JSON))
	router.Use(Handler(key))
	router.Get("/protected", func(c *routing.Context) error {
		user := CurrentUser(c.Request.Context())
		if user == nil {
			return errors.InternalServerError("no user in context")
		}
		return c.Write(map[string]string{"id": user.GetID(), "name": user.GetName()})
	})
	return router
}

func request(router *routing.Router, authorization string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}

func assertAuthenticated(t *testing.T, res *httptest.ResponseRecorder, id, name string) {
	assert.Equal(t, http.StatusOK, res.Code, res.Body.String())
	var body map[string]string
	assert.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	assert.Equal(t, id, body["id"])
	assert.Equal(t, name, body["name"])
}

func assertUnauthorized(t *testing.T, res *httptest.ResponseRecorder) {
	assert.Equal(t, http.StatusUnauthorized, res.Code, res.Body.String())
	assert.Contains(t, res.Header().Get("WWW-Authenticate"), "Bearer")
}

func TestHandler_ValidRoundTrip(t *testing.T) {
	logger, _ := log.NewForTest()
	s := NewService(testSigningKey, 1, logger)
	token, err := s.Login(nil, "demo", "pass")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// the issued token carries exactly the id, name and exp claims, signed with HS256
	parts := strings.Split(token, ".")
	assert.Len(t, parts, 3)
	var header map[string]interface{}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	assert.NoError(t, err)
	assert.NoError(t, json.Unmarshal(headerBytes, &header))
	assert.Equal(t, "HS256", header["alg"])
	var claims map[string]interface{}
	claimBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	assert.NoError(t, err)
	assert.NoError(t, json.Unmarshal(claimBytes, &claims))
	assert.Equal(t, "100", claims["id"])
	assert.Equal(t, "demo", claims["name"])
	exp, ok := claims["exp"].(float64)
	assert.True(t, ok, "exp claim missing")
	assert.InDelta(t, time.Now().Add(time.Hour).Unix(), int64(exp), 5)
	assert.Len(t, claims, 3)

	// the signature matches a stdlib HS256 signature over the same payload
	mac := hmac.New(sha256.New, []byte(testSigningKey))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	assert.Equal(t, b64url(mac.Sum(nil)), parts[2])

	res := request(newTestRouter(testSigningKey), "Bearer "+token)
	assertAuthenticated(t, res, "100", "demo")
}

func TestHandler_ValidStdlibToken(t *testing.T) {
	res := request(newTestRouter(testSigningKey), "Bearer "+hs256Token(t, validClaims(), testSigningKey))
	assertAuthenticated(t, res, "100", "demo")
}

func TestHandler_Expired(t *testing.T) {
	claims := validClaims()
	claims["exp"] = time.Now().Add(-time.Minute).Unix()
	assertUnauthorized(t, request(newTestRouter(testSigningKey), "Bearer "+hs256Token(t, claims, testSigningKey)))
}

func TestHandler_TamperedSignature(t *testing.T) {
	token := hs256Token(t, validClaims(), testSigningKey)
	parts := strings.Split(token, ".")
	sig := []byte(parts[2])
	if sig[0] == 'A' {
		sig[0] = 'B'
	} else {
		sig[0] = 'A'
	}
	tampered := parts[0] + "." + parts[1] + "." + string(sig)
	assertUnauthorized(t, request(newTestRouter(testSigningKey), "Bearer "+tampered))

	// signature stripped entirely
	assertUnauthorized(t, request(newTestRouter(testSigningKey), "Bearer "+parts[0]+"."+parts[1]+"."))
}

func TestHandler_TamperedPayload(t *testing.T) {
	token := hs256Token(t, validClaims(), testSigningKey)
	parts := strings.Split(token, ".")
	claims := validClaims()
	claims["id"] = "1"
	claims["name"] = "admin"
	tampered := parts[0] + "." + encodeSegment(t, claims) + "." + parts[2]
	assertUnauthorized(t, request(newTestRouter(testSigningKey), "Bearer "+tampered))
}

func TestHandler_WrongKey(t *testing.T) {
	token := hs256Token(t, validClaims(), "some-other-key-that-is-not-the-config-key")
	assertUnauthorized(t, request(newTestRouter(testSigningKey), "Bearer "+token))

	// a token signed with an empty key must not validate either
	assertUnauthorized(t, request(newTestRouter(testSigningKey), "Bearer "+hs256Token(t, validClaims(), "")))
}

func TestHandler_AlgNone(t *testing.T) {
	router := newTestRouter(testSigningKey)
	for _, alg := range []string{"none", "None", "NONE", "nOnE"} {
		header := map[string]interface{}{"alg": alg, "typ": "JWT"}
		signingInput := encodeSegment(t, header) + "." + encodeSegment(t, validClaims())

		// empty signature, as the JWS spec describes for "none"
		assertUnauthorized(t, request(router, "Bearer "+signingInput+"."))
		// no signature segment at all
		assertUnauthorized(t, request(router, "Bearer "+signingInput))
		// alg=none but with a real HS256 signature attached
		mac := hmac.New(sha256.New, []byte(testSigningKey))
		mac.Write([]byte(signingInput))
		assertUnauthorized(t, request(router, "Bearer "+signingInput+"."+b64url(mac.Sum(nil))))
		// alg=none with arbitrary garbage as signature
		assertUnauthorized(t, request(router, "Bearer "+signingInput+".c2lnbmF0dXJl"))
	}
}

func TestHandler_Malformed(t *testing.T) {
	router := newTestRouter(testSigningKey)
	valid := hs256Token(t, validClaims(), testSigningKey)
	parts := strings.Split(valid, ".")
	cases := map[string]string{
		"no header":            "",
		"empty bearer":         "Bearer ",
		"bearer without space": "Bearer" + valid,
		"basic scheme":         "Basic " + base64.StdEncoding.EncodeToString([]byte("demo:pass")),
		"token without scheme": valid,
		"lowercase scheme":     "bearer " + valid,
		"garbage":              "Bearer not-a-jwt",
		"two segments":         "Bearer " + parts[0] + "." + parts[1],
		"four segments":        "Bearer " + valid + ".extra",
		"invalid base64":       "Bearer " + "!!!." + parts[1] + "." + parts[2],
		"header not json":      "Bearer " + b64url([]byte("not json")) + "." + parts[1] + "." + parts[2],
		"claims not json":      "Bearer " + parts[0] + "." + b64url([]byte("not json")) + "." + parts[2],
		"truncated":            "Bearer " + valid[:len(valid)-10],
		"whitespace":           "Bearer " + strings.Replace(valid, ".", " . ", -1),
	}
	for name, authorization := range cases {
		t.Run(name, func(t *testing.T) {
			assertUnauthorized(t, request(router, authorization))
		})
	}
}

func TestHandler_SignedTokenMissingClaim(t *testing.T) {
	router := newTestRouter(testSigningKey)
	cases := map[string]map[string]interface{}{
		"missing id":      {"name": "demo", "exp": time.Now().Add(time.Hour).Unix()},
		"missing name":    {"id": "100", "exp": time.Now().Add(time.Hour).Unix()},
		"no claims":       {},
		"only exp":        {"exp": time.Now().Add(time.Hour).Unix()},
		"id not string":   {"id": 100, "name": "demo", "exp": time.Now().Add(time.Hour).Unix()},
		"name not string": {"id": "100", "name": []string{"demo"}, "exp": time.Now().Add(time.Hour).Unix()},
		"id null":         {"id": nil, "name": "demo", "exp": time.Now().Add(time.Hour).Unix()},
	}
	for name, claims := range cases {
		t.Run(name, func(t *testing.T) {
			res := request(router, "Bearer "+hs256Token(t, claims, testSigningKey))
			// must be a clean 401, never a 500 from a panic on the claim type assertion
			assertUnauthorized(t, res)
		})
	}
}

func TestHandler_NonDefaultSigningMethods(t *testing.T) {
	router := newTestRouter(testSigningKey)
	claims := validClaims()

	// HS384 / HS512 with the correct key: only HS256 is accepted
	assertUnauthorized(t, request(router, "Bearer "+hmacToken(t, map[string]interface{}{"alg": "HS384", "typ": "JWT"}, claims, testSigningKey, sha512.New384)))
	assertUnauthorized(t, request(router, "Bearer "+hmacToken(t, map[string]interface{}{"alg": "HS512", "typ": "JWT"}, claims, testSigningKey, sha512.New)))

	// header claims HS256 but the signature is HS512 over the same key
	assertUnauthorized(t, request(router, "Bearer "+hmacToken(t, map[string]interface{}{"alg": "HS256", "typ": "JWT"}, claims, testSigningKey, sha512.New)))

	// RS256: a properly signed RSA token must be rejected because the middleware only accepts HS256
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
	signingInput := encodeSegment(t, map[string]interface{}{"alg": "RS256", "typ": "JWT"}) + "." + encodeSegment(t, claims)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, digest[:])
	assert.NoError(t, err)
	assertUnauthorized(t, request(router, "Bearer "+signingInput+"."+b64url(sig)))

	// key-confusion variant: alg says RS256 but the signature is an HMAC of the shared key
	assertUnauthorized(t, request(router, "Bearer "+hmacToken(t, map[string]interface{}{"alg": "RS256", "typ": "JWT"}, claims, testSigningKey, sha256.New)))

	// unknown algorithm
	assertUnauthorized(t, request(router, "Bearer "+hmacToken(t, map[string]interface{}{"alg": "XS256", "typ": "JWT"}, claims, testSigningKey, sha256.New)))
	// missing alg header
	assertUnauthorized(t, request(router, "Bearer "+hmacToken(t, map[string]interface{}{"typ": "JWT"}, claims, testSigningKey, sha256.New)))
}

func TestHandler_TimeClaims(t *testing.T) {
	router := newTestRouter(testSigningKey)

	future := validClaims()
	future["iat"] = time.Now().Add(time.Hour).Unix()
	assertUnauthorized(t, request(router, "Bearer "+hs256Token(t, future, testSigningKey)))

	notYet := validClaims()
	notYet["nbf"] = time.Now().Add(time.Hour).Unix()
	assertUnauthorized(t, request(router, "Bearer "+hs256Token(t, notYet, testSigningKey)))

	// iat / nbf in the past are fine
	past := validClaims()
	past["iat"] = time.Now().Add(-time.Hour).Unix()
	past["nbf"] = time.Now().Add(-time.Hour).Unix()
	assertAuthenticated(t, request(router, "Bearer "+hs256Token(t, past, testSigningKey)), "100", "demo")

	// exp of the wrong type is rejected
	badExp := validClaims()
	badExp["exp"] = "tomorrow"
	assertUnauthorized(t, request(router, "Bearer "+hs256Token(t, badExp, testSigningKey)))

	// parity with the previous library: exp is optional, a signed token without it is accepted
	noExp := map[string]interface{}{"id": "100", "name": "demo"}
	assertAuthenticated(t, request(router, "Bearer "+hs256Token(t, noExp, testSigningKey)), "100", "demo")
}

func TestHandler_ExtraClaimsIgnored(t *testing.T) {
	claims := validClaims()
	claims["role"] = "admin"
	claims["sub"] = "someone-else"
	res := request(newTestRouter(testSigningKey), "Bearer "+hs256Token(t, claims, testSigningKey))
	assertAuthenticated(t, res, "100", "demo")
}

func TestService_Login(t *testing.T) {
	logger, _ := log.NewForTest()
	s := NewService(testSigningKey, 1, logger)

	for _, c := range [][2]string{{"demo", "wrong"}, {"", ""}, {"admin", "pass"}, {"Demo", "pass"}, {"demo", "PASS"}} {
		token, err := s.Login(nil, c[0], c[1])
		assert.Empty(t, token, c)
		assert.Error(t, err, c)
		if res, ok := err.(errors.ErrorResponse); assert.True(t, ok, c) {
			assert.Equal(t, http.StatusUnauthorized, res.StatusCode())
		}
	}

	// token expiration honours the configured number of hours
	s = NewService(testSigningKey, 48, logger)
	token, err := s.Login(nil, "demo", "pass")
	assert.NoError(t, err)
	parts := strings.Split(token, ".")
	claimBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	assert.NoError(t, err)
	var claims map[string]interface{}
	assert.NoError(t, json.Unmarshal(claimBytes, &claims))
	assert.InDelta(t, time.Now().Add(48*time.Hour).Unix(), int64(claims["exp"].(float64)), 5)

	// a token issued with one key is rejected by a handler configured with another
	assertUnauthorized(t, request(newTestRouter("a-different-deployment-key"), "Bearer "+token))
}

func TestCurrentUser(t *testing.T) {
	assert.Nil(t, CurrentUser(httptest.NewRequest(http.MethodGet, "/", nil).Context()))
	ctx := WithUser(httptest.NewRequest(http.MethodGet, "/", nil).Context(), "100", "demo")
	user := CurrentUser(ctx)
	if assert.NotNil(t, user) {
		assert.Equal(t, "100", user.GetID())
		assert.Equal(t, "demo", user.GetName())
	}
}

func TestMockAuthHandler(t *testing.T) {
	logger, _ := log.NewForTest()
	router := routing.New()
	router.Use(errors.Handler(logger))
	router.Use(MockAuthHandler)
	router.Get("/protected", func(c *routing.Context) error {
		return c.Write(CurrentUser(c.Request.Context()).GetID())
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header = MockAuthHeader()
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Contains(t, res.Body.String(), "100")

	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}
