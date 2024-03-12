package martian

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/plugins/proxy/martianProxy/martian/modifiers"
	"github.com/google/martian/v3"
	"github.com/google/martian/v3/cors"
	"github.com/google/martian/v3/fifo"
	"github.com/google/martian/v3/har"
	"github.com/google/martian/v3/header"
	"github.com/google/martian/v3/log"
	"github.com/google/martian/v3/martianhttp"
	"github.com/google/martian/v3/martianlog"
	"github.com/google/martian/v3/mitm"
	"github.com/google/martian/v3/servemux"
	"github.com/google/martian/v3/verify"

	_ "github.com/google/martian/v3/body"
	_ "github.com/google/martian/v3/cookie"
	_ "github.com/google/martian/v3/failure"
	_ "github.com/google/martian/v3/martianurl"
	_ "github.com/google/martian/v3/method"
	_ "github.com/google/martian/v3/pingback"
	_ "github.com/google/martian/v3/port"
	_ "github.com/google/martian/v3/priority"
	_ "github.com/google/martian/v3/querystring"
	_ "github.com/google/martian/v3/skip"
	_ "github.com/google/martian/v3/stash"
	_ "github.com/google/martian/v3/static"
	_ "github.com/google/martian/v3/status"
)

const (
	REQ  string = "request"
	RESP        = "response"
)

var (
	ErrNoCert = errors.New("No certificate provided, tls proxy won't working")
)

func WithMitmCertsFile(validity time.Duration, name string, organization string, verify bool, cert string, key string, cors bool) ProxyOption {
	return func(p *Proxy) error {
		tlsc, err := tls.LoadX509KeyPair(cert, key)
		if errors.Is(err, os.ErrNotExist) {
			log.Errorf("cert or key  path -> %v, generate certificate", err)
			return WithMitmCertsGenerated(validity, name, organization, verify, cors)(p)
		}
		priv := tlsc.PrivateKey
		x509c, err := x509.ParseCertificate(tlsc.Certificate[0])
		if err != nil {
			return fmt.Errorf("parsing cert failed: %w", err)
		}
		return WithMitm(validity, organization, verify, x509c, priv, cors)(p)

	}
}

func WithMitmCertsGenerated(validity time.Duration, name string, organization string, verify bool, cors bool) ProxyOption {
	return func(p *Proxy) error {
		ca, privKey, err := mitm.NewAuthority(name, organization, validity)
		if err != nil {
			return fmt.Errorf("failed to generated cert key pair")
		}
		return WithMitm(validity, organization, verify, ca, privKey, cors)(p)
	}
}
func WithMitm(validity time.Duration, organization string, verify bool, x509c *x509.Certificate, privKey interface{}, cors bool) ProxyOption {
	return func(p *Proxy) error {
		if x509c != nil && privKey != nil {
			mc, err := mitm.NewConfig(x509c, privKey)
			if err != nil {
				log.Infof("Could not setup mitm -> %v", err)
				return err
			}
			mc.SetValidity(validity)
			mc.SetOrganization(organization)
			mc.SkipTLSVerify(verify)
			p.martian.SetMITM(mc)
			p.mc = mc
			ah := martianhttp.NewAuthorityHandler(x509c)
			p.configure("/authority.cer", ah, cors)
			return nil
		} else {
			return errors.New("missing certificate")
		}
	}
}

func WithEnpointConfiguration(corsEnabled bool, transferToMQTTSettings bool) ProxyOption {
	return func(p *Proxy) error {

		m := martianhttp.NewModifier()
		p.stack.AddRequestModifier(m)
		p.stack.AddResponseModifier(m)
		var mHandler http.Handler = m
		// if cl, err := logger.NewClientInstance(mqttlib.MQTTConfig{}); err == nil && transferToMQTTSettings {
		// 	mHandler = logger.NewMQTTWriter(cl, "/settings").Middleware(m, "POST")
		// } else {
		// 	log.Errorf("initializing middleware for POST configure settings : %v\n", err)
		// }
		p.configure("/configure", mHandler, corsEnabled)
		// Verify assertions expose an endpoint to verify the `martianhttp.Modifier` set previously
		vh := verify.NewHandler()
		vh.SetRequestVerifier(m)
		vh.SetResponseVerifier(m)
		p.configure("/verify", vh, corsEnabled)

		// Reset verifications.
		rh := verify.NewResetHandler()
		var rhHandler http.Handler = rh
		// if cl, err := logger.NewClientInstance(mqttlib.MQTTConfig{}); err == nil && transferToMQTTSettings {
		// 	rhHandler = logger.NewMQTTWriter(cl, "/settings").Middleware(m, "GET")
		// } else {
		// 	log.Errorf("Error initializing middleware for GET reset settings : %v\n", err)
		// }
		rh.SetRequestVerifier(m)
		rh.SetResponseVerifier(m)
		p.configure("/configure/reset", rhHandler, corsEnabled)
		return nil
	}

}

func WithStdLog() ProxyOption {
	return func(p *Proxy) error {
		logger := martianlog.NewLogger()
		logger.SetDecode(true)
		p.stack.AddRequestModifier(logger)
		p.stack.AddResponseModifier(logger)
		return nil
	}
}

type Modifiers interface {
	martian.RequestModifier
}

func WithModifiers(mod any) ProxyOption {
	return func(p *Proxy) error {
		muxf := servemux.NewFilter(p.mux)
		switch modifier := mod.(type) {
		case martian.RequestResponseModifier:
			muxf.RequestWhenFalse(modifier)
			muxf.ResponseWhenFalse(modifier)
			p.stack.AddRequestModifier(muxf)
			p.stack.AddResponseModifier(muxf)
		case martian.RequestModifier:
			muxf.RequestWhenFalse(modifier)
			p.stack.AddRequestModifier(muxf)
		case martian.ResponseModifier:
			muxf.ResponseWhenFalse(modifier)
			p.stack.AddResponseModifier(muxf)
		default:
			return errors.New("unvalid modifier")
		}
		return nil
	}
}

func WithLogInMem(corsEnabled bool) ProxyOption {
	return func(p *Proxy) error {
		hl := har.NewLogger()
		WithModifiers(hl)(p)
		p.configure("/logs", har.NewExportHandler(hl), corsEnabled)
		p.configure("/logs/reset", har.NewResetHandler(hl), corsEnabled)
		return nil
	}
}

func WithHarWriterLog(writer io.Writer) ProxyOption {
	return WithModifiers(modifiers.NewLogger(writer))
}

func WithHierarchicalModifierEnabled(header string) ProxyOption {
	return func(p *Proxy) error {
		return nil
	}
}

//p.logsInMem(stack)
// p.logWs()

func WithLogLevel(level int) ProxyOption {
	return func(p *Proxy) error {
		log.SetLevel(level)
		return nil
	}
}

type ProxyOption = helper.OptionError[Proxy]
type Proxy struct {
	martian    *martian.Proxy
	stack      *fifo.Group
	mux        *http.ServeMux
	mc         *mitm.Config
	address    string
	apiAddress string
	tlsAddress string
}

func NewProxy(address, tlsAddress, apiAddress string, opt ...ProxyOption) *Proxy {
	p := &Proxy{
		martian:    martian.NewProxy(),
		mux:        http.NewServeMux(),
		address:    address,
		apiAddress: apiAddress,
		tlsAddress: tlsAddress,
	}

	// p.stack, p.fg = httpspec.NewStack("loki-proxy")
	hbhm := header.NewHopByHopModifier()
	p.stack = fifo.NewGroup()
	p.stack.AddRequestModifier(hbhm)
	// topGroup.AddResponseModifier(hbhm)
	p.martian.SetRequestModifier(p.stack)
	p.martian.SetResponseModifier(p.stack)

	for _, o := range opt {
		err := o(p)
		if err != nil {
			log.Errorf("proxy option : %v\n", err)
		}
	}
	return p
}

func (p *Proxy) Close() {
	p.martian.Close()
}

// setUpMITM: Set up MITM for martian proxy

// certificate retrieve certificates from cli or generate one

// configure installs a configuration handler at string.
func (p *Proxy) configure(pattern string, handler http.Handler, corsEnabled bool) {
	if corsEnabled {
		handler = cors.NewHandler(handler)
	}
	p.mux.Handle(pattern, handler)
}

func (p *Proxy) Run(ctx context.Context, enableApi bool) error {

	l, err := net.Listen("tcp", p.address)
	if err != nil {
		log.Errorf("%v", err)
		return err
	}
	go p.martian.Serve(l)
	if p.mc != nil {
		tl, err := net.Listen("tcp", p.tlsAddress)
		if err != nil {
			return err
		}

		go p.martian.Serve(tls.NewListener(tl, p.mc.TLS()))
	} // Start TLS listener for transparent MITM.

	if enableApi {
		lAPI, err := net.Listen("tcp", p.apiAddress)
		if err != nil {
			log.Errorf("%v", err)
			return err
		}
		go http.Serve(lAPI, p.mux)

	}
	slog.Info("martian: starting proxy", "proxyAddr", l.Addr().String())
	<-ctx.Done()
	slog.Info("martian: shutting down")
	return nil

}
