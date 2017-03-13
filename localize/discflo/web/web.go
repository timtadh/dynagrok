package web

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

import (
	"github.com/timtadh/data-structures/errors"
	"github.com/timtadh/getopt"
)

import (
	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/discflo"
	"github.com/timtadh/dynagrok/localize/discflo/web/views"
)

func NewCommand(c *cmd.Config, o *discflo.Options) cmd.Runnable {
	return cmd.Cmd(
	"web",
	`[options]`,
	`
Options
    -h, --help                          view this message
    -l, --listen=<addr>:<port>          what to listen on
                                        default: 0.0.0.0:80
    -a, --assets=<path>                 path to asset dir
    --private-ssl-key=<path>
    --ssl-cert=<path>
`,
	"l:a:",
	[]string{
		"listen=",
		"assets",
		"private-ssl-key=",
		"ssl-cert=",
	},
	func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
		listen := "0.0.0.0:80"
		assets := ""
		ssl_key := ""
		ssl_cert := ""
		for _, oa := range optargs {
			var err error
			switch oa.Opt() {
			case "-l", "--listen":
				listen = oa.Arg()
			case "-a", "--assets":
				assets, err = filepath.Abs(oa.Arg())
				if err != nil {
					return nil, cmd.Errorf(1, "assets path was bad: %v", err)
				}
			case "--private-ssl-key":
				ssl_key, err = filepath.Abs(oa.Arg())
				if err != nil {
					return nil, cmd.Errorf(1, "private-ssl-key path was bad: %v", err)
				}
				_, err = os.Stat(ssl_key)
				if os.IsNotExist(err) {
					return nil, cmd.Errorf(1, "private-ssl-key path does not exist. %v", ssl_key)
				} else if err != nil {
					return nil, cmd.Errorf(1, "private-ssl-key path was bad: %v", err)
				}
			case "--ssl-cert":
				log.Println("ssl-cert", oa.Arg())
				ssl_cert, err = filepath.Abs(oa.Arg())
				if err != nil {
					return nil, cmd.Errorf(1, "ssl-cert path was bad: %v", err)
				}
				_, err = os.Stat(ssl_cert)
				if os.IsNotExist(err) {
					return nil, cmd.Errorf(1, "ssl-cert path does not exist. %v", ssl_cert)
				} else if err != nil {
					return nil, cmd.Errorf(1, "ssl-cert path was bad: %v", err)
				}
			default:
				return nil, cmd.Errorf(1, "Unknown flag '%v'\n", oa.Opt())
			}
		}

		if assets == "" {
			return nil, cmd.Errorf(1, "You must supply a path to the assets")
		}

		if (ssl_key == "" && ssl_cert != "") || (ssl_key != "" && ssl_cert == "") {
			return nil, cmd.Errorf(1, "To use ssl you must supply key and cert")
		}

		handler, err := views.Routes(c, o, assets)
		if err != nil {
			return nil, cmd.Err(1, err)
		}

		server := &http.Server{
			Addr: listen,
			Handler: handler,
			ReadTimeout: 1 * time.Second,
			WriteTimeout: 1 * time.Second,
			MaxHeaderBytes: http.DefaultMaxHeaderBytes,
			TLSConfig: nil,
			TLSNextProto: nil,
			ConnState: nil,
			ErrorLog: nil,
		}

		errors.Logf("INFO", "serving @ %v", server.Addr)
		if ssl_key == "" {
			err := server.ListenAndServe()
			if err != nil {
				return nil, cmd.Err(2, err)
			}
		} else {
			log.Println(ssl_cert, ssl_key)
			err := server.ListenAndServeTLS(ssl_cert, ssl_key)
			if err != nil {
				return nil, cmd.Err(2, err)
			}
		}
		return nil, nil
	})
}
