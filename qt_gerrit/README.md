# introduction
qt_gerrit is a bot for interacting with Qt's web services (codereview & JIRA)
from IRC.

# How to run

Set GERRIT_USER to your Gerrit login, GERRIT_PRIVATE_KEY to the
path to your SSH private key for Gerrit

Also set NICKSERV_USER and NICKSERV_PASS to your NickServ credentials.

Set IRC_SERVER to the hostname:port you want to connect to (e.g.
irc.freenode.net:6667)

Set IRC_CHANNELS to a list of channels you want the bot to join, comma
separated, e.g:
#qt-gerrit,#qt-labs

Set GERRIT_CHANNEL to the channel you want to publish Gerrit activity to.

Remember to uncomment aes128cbcID in golang.org/x/crypto/ssh/cipher.go,
until Gerrit finally gets upgraded.
