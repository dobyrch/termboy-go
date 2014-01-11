# Maintainer: Douglas Christman <DouglasChristman [at] gmail [dot] com>
pkgname=termboy
pkgver=0.1.0
pkgrel=1
pkgdesc="A Nintendo Game Boy emulator for the Linux console"
arch=('i686' 'x86_64')
url="https://github.com/dobyrch/termboy-go"
license=('MIT')
makedepends=('go')
source=("https://github.com/dobyrch/$pkgname-go/archive/v$pkgver.tar.gz")
md5sums=('3a061d9f5f2e1be420f284a1ebb5d9a2')

prepare() {
  # Since the source code uses remote import paths, the directory structure
  # needs to be modified so 'go build' can find what it needs
  mkdir -p "src/github.com/dobyrch"
  mv "$pkgname-go-$pkgver" "src/github.com/dobyrch/$pkgname-go"
}

build() {
  cd "src/github.com/dobyrch/$pkgname-go"
  GOPATH="$srcdir" go build
}

package() {
  cd "src/github.com/dobyrch/$pkgname-go"
  install -Dm 755 "$pkgname-go" "$pkgdir/usr/bin/$pkgname-go"
  install -Dm 755 "$pkgname" "$pkgdir/usr/bin/$pkgname"
  install -Dm 644 LICENSE "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
}
