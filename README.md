# ffgh = fzf + gh

Quick access to GitHub PRs from terminal.

## How to use

Either run sync in terminal:

```bash
ffgh-bin -v sync
```

Or run sync from crontab every minute:

```crontab
* * * * * ffgh-bin -v sync -once
```

To run the UI run `./ffgh`.

ffgh **requires** [`fzf`][ref_fzf] and [`gh` CLI][ref_gh].

[ref_fzf]:https://github.com/junegunn/fzf
[ref_gh]:https://cli.github.com/


## Key bindings

* enter - Open all the selected PRs in the browser.
* ctrl-r - Mark as read without opening (does not work with multi-select), mute and unmute.
* ctrl-n - Add note.
* tab - Multi-select.


## xbar

You can use [`ffgh_xbar_plugin.10s.sh`](ffgh_xbar_plugin.10s.sh) as [xbar][ref_xbar] plugin. The muted PRs are ignored
by xbar. If the xbar shows `GH err!` it means that the state is out of sync. Check if synchronization is running.

[ref_xbar]:https://github.com/matryer/xbar


## conifig

You can define a config with GitHub queries. Run ffzf -h to see the default config.


# TODO
* Configure color styles (for black and white terminal)
* Mark as "unread" (for further reading) or mark as "read"
* Do not show the authored new PRs as new, mark them as read automatically.
* Show help with ctrl-?
