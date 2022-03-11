package bench

import (
	"encoding/json"
	"github.com/ory/x/httpx"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func BenchmarkClientGzip(b *testing.B) {
	for _, batchSize := range []int{250, 10000, 100000} {
		b.Run("batchSize="+strconv.Itoa(batchSize), func(b *testing.B) {
			for _, batchBytes := range []int{500 * 1000, 5 * 1000 * 1000, 50 * 1000 * 1000} {
				b.Run("batchBytes="+strconv.Itoa(batchBytes), func(b *testing.B) {
					for _, level := range []int{0, 3, 6, 9} {
						b.Run("compression="+strconv.Itoa(level), func(b *testing.B) {
							var compressedLength int64
							var realLength int64
							var messages int
							server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
								httpx.NewCompressionRequestReader(nil).ServeHTTP(w, r, func(w http.ResponseWriter, r *http.Request) {
									var v json.RawMessage
									if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
										panic(err)
									}
									realLength += int64(len(v))
									pi, _ := strconv.ParseInt(r.Header.Get("Content-Length"), 10, 64)
									compressedLength += pi
									messages++
								})
							}))
							defer server.Close()

							client, _ := analytics.NewWithConfig("h97jamjwbh", analytics.Config{
								GzipCompressionLevel: level,
								Endpoint:             server.URL,
								Verbose:              false,
								Logger:               b,
								BatchMaxSize:         uint(batchBytes),
								BatchSize:            batchSize,
							})
							defer client.Close()

							b.ResetTimer()
							for n := 0; n < b.N; n++ {
								require.NoError(b, client.Enqueue(analytics.Track{
									Event:  "Download" + strconv.Itoa(n),
									UserId: "123456" + strconv.Itoa(n),
									Properties: analytics.Properties{
										"application": "Segment Desktop",
										"version":     "1.1." + strconv.Itoa(n),
										"platform":    "osx",
									},
									Timestamp: time.Now(),
								}))
							}

							b.Logf("Message length %d: %d of %d (%.02f) with %d messages and %.2f of %.2f",
								b.N,
								compressedLength, realLength, 1-float64(compressedLength)/float64(realLength),
								messages,
								float64(compressedLength)/float64(messages), float64(realLength)/float64(messages))
						})
					}
				})
			}
		})
	}
}
