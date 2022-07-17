# gym-jinni
a service for gym owners, trainers and customers to manage their daily gym
activities

Please join this project to implement what
[vagaro.com](https://www.vagaro.com),
[gymmaster.com](https://www.gymmaster.com) and
[strava.com](https://www.strava.com) provide.  Start from implementing minimum
viable key features, work towards providing other major ones, and eventually
desgin something new.

Click this link to review the initial/ongoing [system design document](
https://www.notion.so/d0ca654781a9407199fe1c1abf6e99c7?v=7d033ab95d3e49f4a0b43f211f970a6e)
on notion.

We are using a modular monolithic architecture for this project, taking the
following directory structure:
```
    gym-jinni/
        |____ Vagrantfile
        |____ setup-alx.sh
        |____ service/
        |         |____ Makefile
        |         |____ class/
        |         |         |____ pb/
        |         |         |____ db/
        |         |         |____ class.go
        |         |____ user/
        |         |         |____ pb/
        |         |         |____ db/
        |         |         |____ user.go
        |         |____ validator/
        |         |         |____ pb/
        |         ......
        |____ ui/
        ......
```

After installing vagrant on the host, run "vagrant up" to set up an archlinux
vm (virtual machine) for further development. Please refer to the setup-alx.sh
to install needed software on the host directly if you don't want to set up
the archlinux vm.

Directory service/ is for developing backend service:
```
    * make setup
        - create docker postgres:alpine
        - create database gj
        - migrate up database schemas
    * make test
        - build cmd/service
        - build cmd/grpcclt
        - test the backend service with grpc client and http client
    * define protobuf files for module $MOD under $MOD/pb/
    * define database schemas and queries for module $MOD under $MOD/db/
    * implement business logic for module $MOD within $MOD/$MOD.go
```
Review the service/Makefile to learn more details.

Directory ui/ is for developing frontend user interface; it was generated with
"flutter create ui".

Feel free to send pull requests or drop me a line through
newstart.infotech@gmail.com. If needed, I would add you into
[gymjinni.slack.com](https://gymjinni.slack.com) to discuss further.
