BINDIR     ?= /usr/local/bin
UNITDIR    ?= /etc/systemd/system
SYSCONFDIR ?= /etc
DEFAULTDIR ?= $(SYSCONFDIR)/default
DESTDIR    ?=
PROGNAME   ?= alertmanager_matrix

all: $(PROGNAME) $(PROGNAME).service

$(PROGNAME): $(shell find . -name "*.go") $(shell find -name "*.tmpl")
	@mkdir -p $(@D)
	go build -o $@ ./cmd/$(PROGNAME)

$(PROGNAME).service: build/init/systemd/$(PROGNAME).service.in
	@sed 's|@BINDIR@|$(BINDIR)|g;s|@DEFAULTDIR@|$(DEFAULTDIR)|g' $< > $@

install: install-prog install-unit install-default

install-prog: $(PROGNAME)
	install -Dm 755 $< -t $(DESTDIR)/$(BINDIR)/

install-unit: $(PROGNAME).service
	install -Dm 644 $< -t $(DESTDIR)/$(UNITDIR)/

install-default: build/misc/$(PROGNAME).default
	install -Dm 600 $< -T $(DESTDIR)/$(DEFAULTDIR)/$(PROGNAME)

sources: sources.tar.gz .version

sources.tar.gz:
	go mod vendor
	tar --exclude-from=.gitignore -caf sources.tar.gz *

.version:
	git describe --tags > .version

rpm: sources
	$(MAKE) -C build/package/rpm

clean:
	rm -f *.tar.gz *.rpm .version
	rm -f $(PROGNAME) $(PROGNAME).service
	rm -rf vendor

.PHONY: clean dist install-unit install-prog install-default install
