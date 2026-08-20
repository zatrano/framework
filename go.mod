module github.com/zatrano/framework

go 1.25.0

require (
	github.com/joho/godotenv v1.5.1
	github.com/redis/go-redis/v9 v9.11.0
	github.com/zatrano/framework/packages/database/driver/sqlite v1.0.0
	github.com/zatrano/framework/packages/mongo v1.0.0
	github.com/zatrano/framework/packages/webauthn v1.0.0
	golang.org/x/crypto v0.55.0
	golang.org/x/sys v0.47.0
	modernc.org/sqlite v1.38.2
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/go-webauthn/webauthn v0.17.4 // indirect
	github.com/go-webauthn/x v0.2.6 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.mongodb.org/mongo-driver v1.17.9 // indirect
	golang.org/x/exp v0.0.0-20250620022241-b7579e27df2b // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	modernc.org/libc v1.66.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace (
	github.com/zatrano/framework/packages/database/driver/mongo => ./packages/database/driver/mongo
	github.com/zatrano/framework/packages/database/driver/mssql => ./packages/database/driver/mssql
	github.com/zatrano/framework/packages/database/driver/mysql => ./packages/database/driver/mysql
	github.com/zatrano/framework/packages/database/driver/oracle => ./packages/database/driver/oracle
	github.com/zatrano/framework/packages/database/driver/pgsql => ./packages/database/driver/pgsql
	github.com/zatrano/framework/packages/database/driver/sqlite => ./packages/database/driver/sqlite
	github.com/zatrano/framework/packages/mongo => ./packages/mongo
	github.com/zatrano/framework/packages/qr => ./packages/qr
	github.com/zatrano/framework/packages/webauthn => ./packages/webauthn
)
