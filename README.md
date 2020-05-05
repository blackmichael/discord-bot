# discord-bot

A simple Discord bot, just for fun on a private server, written in Kotlin.

Deployed on [Heroku](https://www.heroku.com) on a [Free Web Dyno](https://devcenter.heroku.com/articles/free-dyno-hours), uses [cron-jobs](https://cron-job.org/en/) to ping the instance every 30 minutes from 7am-12am daily to keep the application awake.

## Local Development
Install required tools:
```
brew update && brew install heroku && brew install gradle
```

Check that everything builds and (eventual) tests pass
```
./gradlew clean build
```

### Run locally (requires bot token)
Via Gradle:
```
DISCORD_TOKEN=<token> ./gradlew run
```

Via Heroku CLI:
```
echo "DISCORD_TOKEN=<token>" > .env
./gradlew clean build && heroku local
```
The `heroku local` command will run all processes defined in `Procfile`, and uses `.env` to define any necessary environment variables.
