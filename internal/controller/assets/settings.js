/**
 * This is the default settings file provided by Node-RED.
 *
 * It can contain any valid JavaScript code that will get run when Node-RED
 * is started.
 *
 * Lines that start with // are commented out.
 * Each entry should be separated from the entries above and below by a comma ','
 *
 * For more information about individual settings, refer to the documentation:
 *    https://nodered.org/docs/user-guide/runtime/configuration
 **/

const rootUrlPath = (process.env.VIRTUAL_PATH) ? `${process.env.VIRTUAL_PATH}` : '';

module.exports = {
    // the tcp port that the Node-RED web server is listening on
    uiPort: process.env.PORT || 1880,

    // Retry time in milliseconds for MQTT connections
    mqttReconnectTime: 15000,

    // Retry time in milliseconds for Serial port connections
    serialReconnectTime: 15000,

    // The maximum length, in characters, of any message sent to the debug sidebar tab
    debugMaxLength: 1000,

    // By default, the Node-RED UI is available at http://localhost:1880/
    // The following property can be used to specify a different root path.
    httpAdminRoot: `${rootUrlPath}/`,

    // Some nodes, such as HTTP In, can be used to listen for incoming http requests.
    // By default, these are served relative to '/'.
    httpNodeRoot: `${rootUrlPath}/api`,

    // Node-RED credential encryption key (managed by the operator).
    credentialSecret: process.env.NODE_RED_CREDENTIAL_SECRET,

    // Securing Node-RED via OAuth2 / niota
    adminAuth: {
      type: 'strategy',
      strategy: {
        name: 'oauth2',
        label: 'Sign in with niota',
        strategy: require('passport-oauth2').Strategy,
        options: {
          authorizationURL: process.env.OAUTH_AUTH_URL,
          tokenURL: process.env.OAUTH_TOKEN_URL,
          clientID: process.env.OAUTH_CLIENT_ID,
          clientSecret: process.env.OAUTH_CLIENT_SECRET,
          callbackURL: process.env.OAUTH_CALLBACK_URL,
          state: true,
          verify: (token, tokenSecret, profile, cb) => {
            return cb(null, {username: 'admin'});
          }
        }
      },
      users: [
        { username: 'admin', permissions: ['*'] }
      ]
    },

    // Seed Global Context
    functionGlobalContext: {},

    exportGlobalContextKeys: false,

    // Configure the logging output
    logging: {
        console: {
            level: "info",
            metrics: false,
            audit: false
        }
    },

    // Customising the editor
    editorTheme: {
        projects: {
            enabled: false
        }
    }
}
