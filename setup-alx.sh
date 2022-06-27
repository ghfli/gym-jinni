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
pacman -S --needed --noconfirm base-devel go git

if ! installed yay ; then
    echo installing yay, an aur helper...
    sudo -u vagrant curl -O https://aur.archlinux.org/cgit/aur.git/snapshot/yay.tar.gz && \
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
    systemctl start snapd
    ln -sf /var/lib/snapd/snap /snap
    snap install flutter --classic
    sudo -u vagrant flutter sdk-path
fi

echo installing any other optional packages you like...
pacman -S --needed --noconfirm darkhttpd vim screen man-db npm docker protobuf ctags

if ! installed dbdocs ; then
    echo installing dbdocs...
    npm install -g dbdocs
fi
if ! installed dbml2sql ; then
    echo installing dbml2sql...
    npm install -g @dbml/cli
fi

install-db-migrate () {
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
if ! installed migrate ; then
    echo installing migrate...
    install-db-migrate
fi

if ! installed sqlc ; then
    echo installing sqlc...
    sudo -u vagrant go install github.com/kyleconroy/sqlc/cmd/sqlc@latest
    ln -sf /home/vagrant/go/bin/sqlc /usr/local/bin
fi

if ! installed mockgen ; then
    echo installing mockgen...
    sudo -u vagrant go install github.com/golang/mock/mockgen@v1.6.0
    ln -sf /home/vagrant/go/bin/mockgen /usr/local/bin
fi

if ! [ -e /home/vagrant/.vimrc ] ; then
    echo linking /vagrant/.vimrc to /home/vagrant...
    sudo -u vagrant ln -sf /vagrant/.vimrc /home/vagrant
fi

if ! [ -e /home/vagrant/.screenrc ] ; then
    echo linking /vagrant/.screenrc to /home/vagrant...
    sudo -u vagrant ln -sf /vagrant/.screenrc /home/vagrant
fi

if ! [ -e /home/vagrant/.bashrc.tail ] ; then
    echo linking /vagrant/.bashrc.tail to /home/vagrant...
    sudo -u vagrant ln -sf /vagrant/.bashrc.tail /home/vagrant
    sudo -u vagrant echo '[ -f ~/.bashrc.tail ] && . ~/.bashrc.tail' >> /home/vagrant/.bashrc
fi

echo done, remember to fix errors if any.
