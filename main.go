package main

import (
    "fmt"
    "io/ioutil"
    "net/url"
    "net/http"
    "net/http/httputil"
    "path/filepath"
    "encoding/json"
    "math/rand"
    "strings"
    "strconv"
    "regexp"
    "log"
    "bytes"
    "time"
    "os"
    "github.com/garyburd/redigo/redis"
)

type Configuration struct {
    RC_secret     string
    RC_key        string
    RedisHost     string
    ListenAddress string
}

type recaptchaResponse struct {
    Success    bool
    ErrorCodes []string `json:"error-codes"`
}

var _redis redis.Conn

func redisConnect() {
    var err error
    _redis, err = redis.Dial("tcp", _configuration.RedisHost)
    if err != nil {
        panic(err)
    }
}

var _configuration = Configuration{}

func readConfig() {
    file, _ := os.Open("conf.json")
    defer file.Close()
    decoder := json.NewDecoder(file)
    err := decoder.Decode(&_configuration)
    if err != nil {
        panic(err)
    }
}

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
    letterIdxBits = 6                    // 6 bits to represent a letter index
    letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
    letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func randStr(n int) string {
    b := make([]byte, n)
    // A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
    for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
        if remain == 0 {
            cache, remain = rand.Int63(), letterIdxMax
        }
        if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
            b[i] = letterBytes[idx]
            i--
        }
        cache >>= letterIdxBits
        remain--
    }

    return string(b)
}

var email_re = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

func initMirrorRsp(rsp http.ResponseWriter) {
    rsp.Header().Add("Server", "MailHide-Mirror")
}

func getStaticResource(rsp http.ResponseWriter, req *http.Request) {
    initMirrorRsp(rsp)
    fmt.Println("path", req.URL.Path)
    path := filepath.Clean(strings.ToLower(req.URL.Path))
    if !strings.HasPrefix(path, "/static/") {
        path = "/static/index.html"
    }
    fmt.Println("path", path[1:])
    contentType := "text/html"
    if strings.HasSuffix(path, ".css") {
        contentType = "text/css"
    } else if strings.HasSuffix(path, ".js") {
        contentType = "text/javascript"
    }
    content, err := ioutil.ReadFile(path[1:])
    if err != nil {
        content, err = ioutil.ReadFile("static/404.html")
        if err != nil {
            rsp.WriteHeader(500)
            return
        }
        rsp.WriteHeader(404)
        fmt.Fprintln(rsp, strings.Replace(string(content), "RSP_URL", path, 1))
        return
    }
    rsp.Header().Add("Content-Type", contentType)
    fmt.Fprintln(rsp, string(content))
}

// Deprecated due to recaptcha referer check policy
func rewriteProxyBody(rsp *http.Response) (err error) {
    b, err := ioutil.ReadAll(rsp.Body) //Read html
    if err != nil {
        return  err
    }
    err = rsp.Body.Close()
    if err != nil {
        return err
    }
    b = bytes.Replace(b, []byte("google.com"), []byte("recaptcha.net"), -1) // replace html
    body := ioutil.NopCloser(bytes.NewReader(b))
    rsp.Body = body
    fmt.Println(body)
    rsp.ContentLength = int64(len(b))
    rsp.Header.Set("Content-Length", strconv.Itoa(len(b)))
    rsp.Header.Set("Server", "MailHide-Mirror")
    return nil
}

// Deprecated due to recaptcha referer check policy
func getMirroredEmail(rsp http.ResponseWriter, req *http.Request) {
    req.Host = "recaptcha.net"
    target, _ := url.Parse("https://recaptcha.net")
    log.Printf("proxying: %s", req.URL)
    pxy := httputil.NewSingleHostReverseProxy(target)
    pxy.ModifyResponse = rewriteProxyBody
    pxy.ServeHTTP(rsp, req)
}

func viewEmail(rsp http.ResponseWriter, req *http.Request) {
    initMirrorRsp(rsp)
    key := req.FormValue("key")
    if req.Method == "POST" { // View Result
        // Check reCAPTCHA
        recaptcha := req.FormValue("g-recaptcha-response")
        client := &http.Client{Timeout: 5 * time.Second}
        resp, err := client.PostForm("https://www.google.com/recaptcha/api/siteverify",
            url.Values{"secret": {_configuration.RC_secret}, "response": {recaptcha}})
        if err != nil {
           fmt.Fprintf(rsp, "Error")
           rsp.WriteHeader(500)
           return
        }
        defer resp.Body.Close()
        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
           fmt.Fprintf(rsp, "Error")
           rsp.WriteHeader(500)
           return
        }

        r := new(recaptchaResponse)

        err = json.Unmarshal(body, r)
        if err != nil {
            fmt.Fprintf(rsp, "Error")
            rsp.WriteHeader(500)
            return
        }
        if !r.Success {
            fmt.Fprintf(rsp, "Invalid reCAPTCHA response")
            rsp.WriteHeader(403)
            return
        }

        content, err := ioutil.ReadFile("static/result.html")
        if err != nil {
            rsp.WriteHeader(500)
            return
        }
        email, err := redis.String(_redis.Do("GET", "mailhide_" + key))
        if err != nil {
            rsp.WriteHeader(404)
            fmt.Fprintln(rsp, "Not found")
            return
        }
        fmt.Fprintln(rsp, strings.Replace(string(content), "RSP_EMAIL", email, 2))
        return

    }
    // Verification Page
    content, err := ioutil.ReadFile("static/show.html")
    if err != nil {
        rsp.WriteHeader(500)
        return
    }
    fmt.Fprintln(rsp, strings.Replace(string(content), "RC_KEY", _configuration.RC_key, 1))
}

func saveEmail(rsp http.ResponseWriter, req *http.Request) {
    initMirrorRsp(rsp)
    email := req.FormValue("email")
    if !email_re.MatchString(email) {
        rsp.WriteHeader(400)
        return
    }
    key, err := redis.String(_redis.Do("GET", "mailhide_" + email))

    if err != nil {
        key = randStr(16)
        _, err = _redis.Do("SET", "mailhide_" + key, email)
        if err != nil {
            rsp.WriteHeader(500)
            return
        }
        _, err = _redis.Do("SET", "mailhide_" + email, key)
        if err != nil {
            rsp.WriteHeader(500)
            return
        }
    }
    content, err := ioutil.ReadFile("static/save.html")
    if err != nil {
        rsp.WriteHeader(500)
        return
    }
    output := strings.Replace(string(content), "RSP_EMAIL", email, 1)
    output = strings.Replace(output, "RSP_KEY", key, 5)
    output = strings.Replace(output, "RSP_EMAIL_FIRST_LETTER", email[0:1], 2)
    output = strings.Replace(output, "RSP_EMAIL_DOMAIN", strings.Split(email, "@")[1], 2)
    fmt.Fprintln(rsp, output)
}

func main() {
    rand.Seed(time.Now().UnixNano())
    readConfig()
    redisConnect()
    http.HandleFunc("/", getStaticResource)
    http.HandleFunc("/d", viewEmail)
    http.HandleFunc("/save", saveEmail)
    //http.HandleFunc("/d", getMirroredEmail)
    //http.HandleFunc("/recaptcha/", getMirroredEmail)
    // http.HandleFunc("/static/", getStaticResource)
    err := http.ListenAndServe(_configuration.ListenAddress, nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}
