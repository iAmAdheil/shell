# 24: Live PTY resize to smallest connected viewport

**What to build:** The one shared PTY is sized to whichever connected User currently has the smallest terminal viewport. The size updates live as Users join, leave, or resize their own browser window.

**Blocked by:** 22

**Status:** ready-for-agent

- [ ] With two Users connected at different viewport sizes, the shared PTY is sized to the smaller of the two.
- [ ] When the User with the smaller viewport disconnects, the PTY resizes live to match the next-smallest remaining viewport.
- [ ] When any connected User resizes their browser window, the PTY resizes live if their new size changes which viewport is smallest.
- [ ] A newly joining User with a smaller viewport than everyone else already connected triggers an immediate PTY resize down to their size.
