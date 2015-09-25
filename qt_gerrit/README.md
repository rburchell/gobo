# introduction
qt_gerrit is a bot for interacting with Qt's web services (codereview & JIRA)
from IRC.

# setup
* go get
* Remember to uncomment aes128cbcID in golang.org/x/crypto/ssh/cipher.go,
  until Gerrit finally gets upgraded.
* go build

# how to run

You need a bunch of environment variables set:

* GERRIT_USER: your Gerrit username
* GERRIT_PRIVATE_KEY: the path to your SSH private key for Gerrit
* NICKSERV_USER: NickServ username
* NICKSERV_PASS: NickServ password
* IRC_SERVER: hostname:port to the IRC server you want to announce on
* IRC_CHANNELS: a comma-separated list of channels you want the bot in,
  e.g. #qt-labs,#qt-gerrit
* GERRIT_CHANNEL: the channel you want to publish Gerrit activity to.
