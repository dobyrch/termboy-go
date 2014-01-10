# Maintainer: Douglas Christman <DouglasChristman [at] gmail [dot] com>
pkgname=termboy
pkgver=0.1.0
pkgrel=1
pkgdesc="A Nintendo Game Boy emulator for the Linux console"
arch=('x86_64')
url="http://github.com/dobyrch/termboy-go"
license=('MIT')
depends=('bash')
makedepends=('go')
source=(http://github.com/dobyrch/$pkgname-go/archive/v$pkgver.tar.gz)
md5sums=('3a061d9f5f2e1be420f284a1ebb5d9a2')

build() {
  cd $pkgname-go-$pkgver
  go build
}

package() {
  cd $pkgname-go-$pkgver
  install -Dm 755 $pkgname-go-$pkgver $pkgdir/usr/bin/$pkgname-go
  install -Dm 755 $pkgname $pkgdir/usr/bin/$pkgname
  install -Dm 644 LICENSE $pkgdir/usr/share/licenses/$pkgname/LICENSE
}
