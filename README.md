# YeetBot
This bot yeets inactive users from your server, the following commands allow you to modify this behaviour.

Activity is based on message creation and on voice state events (joining voice channel, moving, leaving, etc.).  
The bot will warn you on the halfway mark as well as the final day before you get kicked by default.

### Notes
On first join remember to run `!yeet forceadd` so that yeetbot can scan through all the users it needs to keep track of.


The bot will only respond to the _**owner of the server**_, that being the person who created the server or the person who was appointed as the new owner in the server settings. To prevent the bot from spamming in a channel when non-owners try to send commands the bot will simply not reply.  
You can make people immune to getting kicked by running `!yeet immune (mention person)`

## Commands
```
 - help               | Shows this help dialog
 - timeout            | Gets the timeout (in days) before a user gets kicked
 - timeout (days)     | Sets the timeout (in days) before a user gets kicked
 - warntimeout (days) | Sets the timeout (in days) before a user gets warned, set to -1 to show the warning at the halfway mark
 - warntimeout        | Gets the timeout (in days) before a user gets warned
 - kickmsg            | Gets the message displayed when a user gets kicked
 - kickmsg (msg)      | Sets the message displayed when a user gets kicked
 - warnmsg            | Gets the message displayed when a user gets warned
 - warnmsg (msg)      | Sets the message displayed when a user gets warned
 - isimmune (mention) | Gets the user's immunity to being kicked
 - immune (mention)   | Toggles the user's immunity to being kicked
 - forceadd           | Forces all users (that make sense) to be added to yeetbots internal timing list
 - (mention)          | Forcefully yeets that person with a dumb message, you evil tater
```
