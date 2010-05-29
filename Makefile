include $(GOROOT)/src/Make.$(GOARCH)

TARG=ctar
GOFILES=\
	$(TARG).go\

include $(GOROOT)/src/Make.cmd
