# Implementation Order

1. Add harness package with:
   - registry.go
   - policy.go
   - prompt_builder.go
   - evidence.go
   - validator.go
   - trace.go
   - service.go

2. Replace chat prompt construction with prompt builder

3. Replace tool switch exposure with registry metadata

4. Add policy checks before dispatch

5. Add bounded advisory loop

6. Add validator pass and retry/refuse logic

7. Add trace persistence or log sink

8. Expose trace id in assistant responses

9. Add tests:
   - forbidden tool rejected
   - weak evidence forces uncertainty
   - unsupported live-data claim rejected
   - no approval/execution language allowed

10. Roll out in research mode first, then paper, then live advisory
