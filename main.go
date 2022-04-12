package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

func main() {

	// Construct a new API object and get env api key
	api, err := cloudflare.NewWithAPIToken(os.Getenv("CF_API_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}
	// Most API calls require a Context
	ctx := context.Background()

	// GET all id domain
	z, err := api.ListZones(ctx)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Printf("%#v\n", z)
	for _, zi := range z {
		if zi.Status == "active" {

			//fmt.Printf("  %v:\n    id: %v\n", zi.Name, zi.Name)
			site := zi.Name
			fmt.Printf("=>Test site https://%v\n", site)
			// httptrace
			// test to http get
			// Timeout to 10 sec
			client := &http.Client{
				Timeout: time.Second * 20,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					if len(via) > 2 {
						// Check SSL flexible
						ssl, err := api.ZoneSSLSettings(ctx, zi.ID)
						if err != nil {
							log.Fatal(err)
						}
						if ssl.Value == "flexible" {
							fmt.Println("SSL/TLS flexible, change...")
							res, err := api.UpdateZoneSSLSettings(ctx, zi.ID, "full")
							if err != nil {
								log.Fatal(err)
							}
							fmt.Printf("Modify domain %v SSL to %v \n", zi.Name, res.Value)
							time.Sleep(10)
							//return nil
							return fmt.Errorf("change SSL, max 2 hops\n")
						}
					} else if len(via) == 1 {
						fmt.Println("Redirect ===> ")
					}
					return nil
				},
			}
			req, _ := http.NewRequest("GET", "https://"+site, nil)
			// added a user agent to disguise itself as a browser :)
			req.Header.Add("user-agent", `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.106 Safari/537.36`)
			// code copy https://blog.golang.org/http-tracing
			var start, connect, dns, tlsHandshake time.Time

			trace := &httptrace.ClientTrace{
				DNSStart: func(dsi httptrace.DNSStartInfo) { dns = time.Now() },
				DNSDone: func(ddi httptrace.DNSDoneInfo) {
					fmt.Printf("DNS Done: %v\n", time.Since(dns))
				},

				TLSHandshakeStart: func() { tlsHandshake = time.Now() },
				TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
					fmt.Printf("TLS Handshake: %v\n", time.Since(tlsHandshake))
				},

				ConnectStart: func(network, addr string) { connect = time.Now() },
				ConnectDone: func(network, addr string, err error) {
					fmt.Printf("Connect time: %v\n", time.Since(connect))
				},

				GotFirstResponseByte: func() {
					fmt.Printf("Time from start to first byte: %v\n", time.Since(start))
				},
			}

			req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
			start = time.Now()
			if _, err := http.DefaultTransport.RoundTrip(req); err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Total time: %v\n", time.Since(start))
			// end copy
			//
			resp, err := client.Do(req)
			if err != nil {
				log.Fatal(err)
			}
			// to clean up resources
			defer resp.Body.Close()
			fmt.Printf("Respons status code: %v\n", resp.StatusCode)
			fmt.Println("--------------------------------------")

		}
	}

}
