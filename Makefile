# Build configuration
GO := go
BINARY_NAME := restreminder
SRC_DIR := ./src

# Version control (dynamically pull from git tags, fallback to 1.0.0, prepended with 0.0.0~ if starting with a letter)
VERSION ?= $(shell (git describe --tags --always --dirty 2>/dev/null || echo "1.0.0") | sed 's/^v//' | sed -E 's/^([^0-9])/0.0.0~\1/')

# Default target
all: dev

LDFLAGS := -X main.version=$(VERSION)

# Local development build
dev:
	CGO_ENABLED=1 $(GO) build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) $(SRC_DIR)

# Optimized production build
prod:
	CGO_ENABLED=1 $(GO) build -ldflags="-s -w $(LDFLAGS)" -o $(BINARY_NAME) $(SRC_DIR)

# Build Debian package
deb: prod
	@echo "Building Debian package for restreminder v$(VERSION)..."
	mkdir -p build/deb/DEBIAN
	mkdir -p build/deb/usr/bin
	mkdir -p build/deb/usr/share/applications
	mkdir -p build/deb/usr/share/pixmaps
	
	# Copy binary
	cp $(BINARY_NAME) build/deb/usr/bin/
	
	# Copy icon
	cp assets/icon-512.png build/deb/usr/share/pixmaps/restreminder.png
	
	# Generate .desktop shortcut
	echo "[Desktop Entry]" > build/deb/usr/share/applications/restreminder.desktop
	echo "Name=Rest Reminder" >> build/deb/usr/share/applications/restreminder.desktop
	echo "Comment=Desktop Pomodoro rest reminder application" >> build/deb/usr/share/applications/restreminder.desktop
	echo "Exec=/usr/bin/restreminder" >> build/deb/usr/share/applications/restreminder.desktop
	echo "Icon=restreminder" >> build/deb/usr/share/applications/restreminder.desktop
	echo "Terminal=false" >> build/deb/usr/share/applications/restreminder.desktop
	echo "Type=Application" >> build/deb/usr/share/applications/restreminder.desktop
	echo "Categories=Utility;" >> build/deb/usr/share/applications/restreminder.desktop
	echo "StartupWMClass=com.restreminder.app" >> build/deb/usr/share/applications/restreminder.desktop
	
	# Generate control file
	echo "Package: restreminder" > build/deb/DEBIAN/control
	echo "Version: $(VERSION)" >> build/deb/DEBIAN/control
	echo "Section: utils" >> build/deb/DEBIAN/control
	echo "Priority: optional" >> build/deb/DEBIAN/control
	echo "Architecture: amd64" >> build/deb/DEBIAN/control
	echo "Maintainer: RestReminder Developer" >> build/deb/DEBIAN/control
	echo "Description: Native desktop Pomodoro rest reminder application." >> build/deb/DEBIAN/control
	
	# Compile package
	dpkg-deb --build build/deb restreminder_$(VERSION)_amd64.deb
	
	# Cleanup temp build files
	rm -rf build/deb
	@echo "Debian package built: restreminder_$(VERSION)_amd64.deb"

# Cleanup build outputs
clean:
	rm -f $(BINARY_NAME)
	rm -rf build
	rm -f restreminder_*.deb

.PHONY: all dev prod deb clean
