#!/bin/bash
set -e # exit immediately if a command fails

echo setting up archlinux for gym-jinni dev...

echo upgrading the whole system...
pacman -Syu --needed --noconfirm

echo installing base-devel, go and git for yay...
pacman -S --needed --noconfirm base-devel go git

if which yay ; then
    echo yay has been installed.
else
    echo installing yay, an aur helper...
    sudo -u vagrant curl -O https://aur.archlinux.org/cgit/aur.git/snapshot/yay.tar.gz && \
    sudo -u vagrant tar xvf yay.tar.gz && \
    cd yay && sudo -u vagrant makepkg -fsrCc && pacman --noconfirm -U yay*.zst
fi

if which snap ; then
    echo snapd has been installed.
else
    echo installing snapd with yay...
    sudo -u vagrant yay -S --needed --noconfirm snapd
fi

if which flutter ; then
    echo flutter has been installed.
else
    echo installing flutter with snapd...
    systemctl start snapd
    ln -fs /var/lib/snapd/snap /snap
    snap install flutter --classic
    sudo -u vagrant flutter sdk-path
fi

echo installing any other optional packages you like...
pacman -S --needed --noconfirm darkhttpd vim screen man-db

echo done.
