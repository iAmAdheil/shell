Type: grilling
Status: resolved

## Question

One PTY is shared by every User in a Session, but each User's browser window can be a different size, and only one size can be sent to the PTY at a time. What size wins?

## Answer

The PTY is sized to whichever connected User currently has the smallest viewport, and resizes live as Users join, leave, or resize their own window. Not a fixed size, and not fixed to the creating User's window.
