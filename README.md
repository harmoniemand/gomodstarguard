# gomodstarguard

heavily inspired by [gomodguard](https://github.com/ryancurrah/gomodguard)

A golang linter that checks dependencies from github for their stars and warns if they are below a certain threshold.


    # .gomodstarguard.yml
    warn: 10
    error: 5
    exeptions:
      - repository: github.com/harmoniemand/gomodstars
        reason: sadly this repo does not have that many stars yet
