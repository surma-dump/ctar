include $(GOROOT)/src/Make.$(GOARCH)

TARG=epd
GOFILES=\
	epd.go\

include $(GOROOT)/src/Make.cmd
