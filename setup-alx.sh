#!/bin/bash
set -e # exit immediately if a command fails
echo PATH=$PATH PWD=$(pwd) whoami=$(whoami)

installed () {
    type $1
}

echo setting up archlinux for gym-jinni dev...

echo upgrading the whole system...
pacman -Syu --needed --noconfirm

echo setting timezone...
timedatectl set-timezone America/Vancouver

echo installing base-devel, go and git for yay...
pacman -S --needed --noconfirm base-devel fuse go git squashfuse screen vim

if ! installed yay ; then
    echo installing yay, an aur helper...
    sudo -u vagrant curl -O \
	    https://aur.archlinux.org/cgit/aur.git/snapshot/yay.tar.gz && \
        sudo -u vagrant tar xvzf yay.tar.gz && \
        cd yay && \
        sudo -u vagrant makepkg -fsrCc && \
        pacman --noconfirm -U yay*.zst
fi

if ! installed snap ; then
    echo installing snapd with yay...
    sudo -u vagrant yay -S --needed --noconfirm snapd
fi

if ! installed flutter ; then
    echo installing flutter with snapd...
    echo if it failed, try to reload the vm again, i.e.,
    echo vagrant reload --provision
    systemctl start snapd
    ln -sf /var/lib/snapd/snap /snap
    snap install flutter --classic
    sudo -u vagrant flutter sdk-path
fi

echo installing any other optional packages you like...
# pacman -D --noconfirm npm && rm -rf /usr/lib/node_modules/npm
pacman -S --needed --noconfirm darkhttpd man-db npm docker \
    protobuf ctags github-cli

if ! installed dbdocs ; then
    echo installing dbdocs...
    npm install -g dbdocs
fi
if ! installed dbml2sql ; then
    echo installing dbml2sql...
    npm install -g @dbml/cli
fi

install_db_migrate () {
    installed migrate && return
    echo installing migrate...
    version=${1:-v4.15.2}
    platform=${2:-linux}
    tarfn=migrate.$platform-amd64.tar.gz
    url=https://github.com/golang-migrate/migrate/releases/download/$version/$tarfn
    dst=/home/vagrant/migrate-$version
    echo downloading $url to $dst...
    sudo -u vagrant mkdir -p $dst && cd $dst && \
        sudo -u vagrant curl -OL $url && \
        sudo -u vagrant tar xvzf $tarfn
    echo linking $dst/migrate to /usr/local/bin/...
    ln -sf $dst/migrate /usr/local/bin
}
install_db_migrate

go_install () {
    url=$1
    cmd=$(basename $url)
    installed $cmd && return
    version=${2:-latest}
    echo installing $cmd@$version...
    sudo -u vagrant go install $url@$version
    ln -sf /home/vagrant/go/bin/$cmd /usr/local/bin
}
go_install github.com/kyleconroy/sqlc/cmd/sqlc
go_install google.golang.org/protobuf/cmd/protoc-gen-go
go_install google.golang.org/grpc/cmd/protoc-gen-go-grpc
go_install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
go_install github.com/mwitkow/go-proto-validators/protoc-gen-govalidators
go_install github.com/golang/mock/mockgen v1.6.0

if ! [ -e /home/vagrant/.bashrc.tail ] ; then
    sudo -u vagrant echo '[ -f ~/.bashrc.tail ] && . ~/.bashrc.tail' >> /home/vagrant/.bashrc
fi

for rc in .vimrc .screenrc .bashrc.tail ; do
    if ! [ -e /home/vagrant/$rc ] ; then
        echo linking /vagrant/$rc to /home/vagrant...
        sudo -u vagrant ln -sf /vagrant/$rc /home/vagrant
    fi
done

if ! installed brew ; then
    echo installing brew...
    sudo -u vagrant /bin/bash -c \
	    "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    sudo -u vagrant echo 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"' >> /home/vagrant/.bash_profile
    eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
fi

if ! installed buf ; then
    echo installing buf...
    sudo -u vagrant brew install bufbuild/buf/buf
fi

if ! installed jq ; then
    echo installing jq...
    curl -fsSL https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64 \
	--output-dir /usr/local/bin -o jq
    chmod +x /usr/local/bin/jq
fi

echo done, remember to fix errors if any.
