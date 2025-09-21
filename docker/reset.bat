docker rm -f kali

#docker run -d --name kali linuxserver/kali-linux

docker run -d --name kali --cap-add=NET_RAW --cap-add=NET_ADMIN linuxserver/kali-linux