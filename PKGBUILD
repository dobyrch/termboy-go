# TODO: Complete and test PKGBUILD (remember to run namcap)
# Maintainer: Douglas Christman <DouglasChristman [at] gmail [dot] com>
pkgname=termboy
pkgver=0.1.0
pkgrel=1
pkgdesc="A Nintendo Game Boy emulator for the Linux console"
arch=('x86_64')
url="http://github.com/dobyrch/termboy-go"
license=('MIT')
groups=()
depends=()
makedepends=('go')
optdepends=()
provides=()
conflicts=()
replaces=()
backup=()
options=()
install=
changelog=
source=($pkgname-$pkgver.tar.gz)
noextract=()
md5sums=() #autofill using updpkgsums

build() {
  cd $pkgname-$pkgver

  ./configure --prefix=/usr
  make
}

package() {
  cd $pkgname-$pkgver

  make DESTDIR="$pkgdir/" install
}.
