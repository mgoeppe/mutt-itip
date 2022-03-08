[![go: build & test](https://github.com/matoubidou/mutt-itip/actions/workflows/go.yaml/badge.svg)](https://github.com/matoubidou/mutt-itip/actions/workflows/go.yaml) ![codecov](https://codecov.io/gh/matoubidou/mutt-itip/branch/main/graph/badge.svg)

# mutt-itip

Enables mutt users to deal with iCal invitations in mutt. The overall process is
defined in [RFC 5546](https://datatracker.ietf.org/doc/html/rfc5546).Currently
only the following features are implemented:

* accept / declined / tentatively accept invitations sending back reply mails to
  organizer. So far local calendar is not updated - would we need to do that?

  Refer to [Section
  3.2.3](https://datatracker.ietf.org/doc/html/rfc5546#section-3.2.3) of the
  RFC.

* cancellation of a invitation deletes corresponding event from your local
  calendar, i.e. the corresponding ics file is deleted from your icsDir.

  Refer to [Section
  3.2.5](https://datatracker.ietf.org/doc/html/rfc5546#section-3.2.5) of the
  RFC.

>Note: Feel free to contribute other features, e.g. counter proposal, decline
>counter proposal, ...
## Installation

Get the binary using go:

```
go get -u github.com/matoubidou/mutt-itip
```

In order to display invitations in mutt add the following lines to your
`mailcap`:

```
text/calendar; mutt-itip format; copiousoutput
application/ics; mutt-itip format; copiousoutput
```

In order to be reply to invitations you need to configure mutt-itip as it needs
to be able to send mails and access local calendars information. Here is a what
the configuration file should look like:

```
cat << EOF > $HOME/.config/mutt-itiprc
mail: <your mail address>
icsDir: <directory where the ics files of your local calendar reside>
smtpAddr: <address of the smpt server to use, e.g. localhost:1025>
smtpUser: <user to use for smtp auth>
smtpPassCmd: <command printing password to use for smtp auth to stdout, e.g. pass youruser>
EOF
```

In order to be able to reply in mutt you might want to use the following macros:

```
bind index,pager i noop
macro index,pager ia '<pipe-entry>mutt-itip accept "Accept iCal invitation"
macro index,pager id '<pipe-entry>mutt-itip decline "Decline iCal invitation"
macro index,pager it '<pipe-entry>mutt-itip tentative "Tentatively accept iCal invitation"
macro index,pager iu '<pipe-entry>mutt-itip update "Update / Delete iCal invitation"
```
