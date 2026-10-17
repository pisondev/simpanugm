# Definition of Ready and Definition of Done | SimpanUGM

## Definition of ready

A story moves to Ready only when:

1. It has an issue with the story text, acceptance criteria, requirement ids, area,
   priority and estimate.
2. Its acceptance criteria are testable as written.
3. Its contracts are known: the endpoints in `openapi.yaml` or a pull request adding them,
   the tables in data-model, the screens in information-architecture.
4. Its dependencies are merged or scheduled earlier in the same sprint.
5. It fits in one sprint. Anything above 8 points is split.

## Definition of done

A story is done only when every line holds.

### Behaviour

- [ ] Every acceptance criterion is met and reachable through the real entry point: the
      page for UI stories, the endpoint for API stories, the job for background stories.
- [ ] Error, empty and loading states of information-architecture section 4 exist for
      every new screen.
- [ ] UI copy is Indonesian, comes from the message catalog and follows ui-guidelines.

### Tests

- [ ] Unit tests cover the logic, including each business rule the story touches.
- [ ] Integration tests cover each boundary the story touches: HTTP, database, storage,
      jobs, inference API.
- [ ] Permission and projection tests are extended for every new endpoint or field.
- [ ] `make check` passes locally and CI is green.
- [ ] The affected end to end journey still passes.

### Quality

- [ ] Lint and type checks are clean, coverage did not drop by more than 2 points without
      a stated reason.
- [ ] No secret, personal data, dataset or weight is committed.
- [ ] Logs added by the story follow observability section 1 and contain no redacted key.
- [ ] Accessibility: axe has no serious or critical finding on new pages, keyboard
      operation checked by hand.

### Documentation

- [ ] Docs that describe the changed behaviour are updated in the same pull request.
- [ ] `CHANGELOG.md` has a line under `Unreleased`.
- [ ] A new decision that is expensive to reverse has an ADR.

### Delivery

- [ ] Reviewed and approved per CONTRIBUTING section 5, squash merged.
- [ ] Deployed to staging by the pipeline and checked there by someone other than the
      author.

## Done for AI work

An experiment, a model version or a fusion version is done when, in addition:

- [ ] The run is reproducible from the repository per evaluation section 10.
- [ ] The run id, config and gate table are in the pull request.
- [ ] Results that fail a gate are recorded in the evaluation report anyway.
- [ ] A model card section exists for anything activated.
