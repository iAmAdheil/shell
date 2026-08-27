# 23: Connected-User roster

**What to build:** Every User in a Session sees a live list of who else is currently connected, identified by the account each User authenticated with. The list updates as Users join and leave.

**Blocked by:** 22

**Status:** ready-for-agent

- [ ] Every connected User sees a roster listing every currently connected User by their authenticated identity (e.g. name/avatar from the OAuth provider), not a made-up nickname.
- [ ] When a new User joins, the roster updates for every already-connected User without a page reload.
- [ ] When a User disconnects, the roster updates for every remaining User without a page reload.
- [ ] A newly joining User sees the same roster the already-connected Users see.
