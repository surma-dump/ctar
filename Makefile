include $(GOROOT)/src/Make.$(GOARCH)

TARG=ctar
GOFILES=\
	ctar.go\

include $(GOROOT)/src/Make.cmd
