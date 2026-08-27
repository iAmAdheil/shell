Type: grilling
Status: resolved

## Question

Can a Session Code be reused after its Session ends (CONTEXT.md: a Session ends the moment its last User disconnects)?

## Answer

No. The code dies with the Session. Opening an ended Session's code shows an error ("this Session has ended"); it does not silently start a new Session under the same code.
