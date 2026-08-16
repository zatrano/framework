package mongo

// Blank-import this module (via `zatrano db:setup --drivers=…,mongo`) so MongoDB
// is linked like other database drivers. Boot wires the client when "mongo" is
// listed in DB_CONNECTIONS.
import _ "github.com/zatrano/framework/packages/mongo"
