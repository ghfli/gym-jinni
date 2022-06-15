#!/bin/bash
set -e # exit immediately if a command fails

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
pacman -S --needed --noconfirm darkhttpd vim screen man-db npm
if ! installed dbdocs ; then
    npm install -g dbdocs
fi
if ! installed dbml2sql ; then
    npm install -g @dbml/cli
fi

install-db-migrate () {
    version=${1:-v4.15.2}
    platform=${2:-linux}
    tarfn=migrate.$platform-amd64.tar.gz
    url=https://github.com/golang-migrate/migrate/releases/download/$version/$tarfn
    dst=/home/vagrant/migrate-$version
    echo download $url to $dst...
    sudo -u vagrant mkdir -p $dst && cd $dst && \
        sudo -u vagrant curl -OL $url && \
        sudo -u vagrant tar xvzf $tarfn
    echo link $dst/migrate to /usr/local/bin/...
    ln -sf $dst/migrate /usr/local/bin
}

if ! installed migrate ; then
    install-db-migrate
fi

echo done. remember to fix any errors manually on the vm.
