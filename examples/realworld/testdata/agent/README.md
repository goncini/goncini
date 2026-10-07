# The agent test

M1's last gate: an agent given only goncini's AGENTS.md adds an endpoint with
validation and a new error case to this app, and hidden tests pass.

To run it:

1. Copy the repository, without `.git`, to a scratch directory.
2. Give a fresh agent the task below, and nothing else.
3. When it is done, copy `edit_comment_test.ego.txt` to the copy's
   `examples/realworld/articles/edit_comment_test.ego`, then run
   `go tool ego generate ./articles/ && go test -run TestEditComment ./articles/`.

Passed on 2026-10-07, at the first attempt, with AGENTS.md's first version,
which then took in the agent's feedback.

## The task

> You are adding a feature to an API built with the goncini framework. Work
> only in the copy. First read AGENTS.md at its root: it is the only
> documentation you may read. You may read and edit the app in
> examples/realworld. Don't read the framework's own source or docs, and
> don't modify anything outside examples/realworld.
>
> Add an endpoint to edit a comment: PUT /api/articles/{slug}/comments/{id},
> with the body {"comment": {"body": "new text"}}.
> - It needs an authenticated user: without a token it answers 401 with
>   {"errors":{"token":["is missing"]}}.
> - The body is required: an empty one answers 422 with
>   {"errors":{"body":["can't be blank"]}}.
> - An unknown article answers 404 {"errors":{"article":["not found"]}}; an
>   unknown comment answers 404 {"errors":{"comment":["not found"]}}; a user
>   who isn't the comment's author gets 403 {"errors":{"comment":["forbidden"]}}.
> - A comment can be edited only within 15 minutes of its creation. After
>   that, the endpoint answers 409 with
>   {"errors":{"comment":["can no longer be edited"]}}, and the comment is
>   unchanged. Make this a new case of the articles feature's error set.
> - On success it answers 200 with {"comment": {...}}, in the same shape as
>   when a comment is created, with an updatedAt of the time of the edit. The
>   edit persists.
> - Use the app's clock or time.Now for times (tests run on a fake clock).
>
> Run `go tool ego generate ./...` and `go test ./...`, and make sure
> everything builds and the existing tests pass.
