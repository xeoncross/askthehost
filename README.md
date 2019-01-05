
# [AskThe.Host](http://askthe.host)

A free, public speaking application for eliciting responses from an audience via text messages.

This system will receive text messages and push them via websockets to the client. When a client visits this URL they will see all the texts sent along with admin controls.

No account is needed, because the presenter can use the frontend controls to delete or ban texts while they show up on the big screen.

# History

For public speaking I wanted a way to:

1) run audience polls without counting hands
2) discuss more intimate questions from the audience without making them speak in front of a large crowd.
3) Give people who aren't conformable with public speaking a way to ask questions

This project has served those purposes for a couple years now. I am finally getting around to open sourcing it. The backend has no persistent store (no database), no unit tests, and no framework. The frontend is written in Angular 1.

This project was written long before [plivo had a Go SDK](https://developers.plivo.com/server-side-sdks/go-sdk/).

Please excuse the layout and structure.

# SMS Gateway

[Nexmo](https://nexmo.com) and [Twilio](https://www.twilio.com) were evaluated, but [Plivo](https://www.plivo.com) was chosen due to the free incoming message support. The only fee is the $0.80/mo charge for each SMS number.

## Install Frontend Assets

    cd js
    npm install
    bower install
    gulp scripts

## Install Go libraries

    go get

## Run backend server

    go build -o smsapp *.go
    nohup ./smsapp >> smsapp.log 2> smserr.log &
