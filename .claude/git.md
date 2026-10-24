# Git 작업 규칙

- main에서 직접 작업하지 않는다. 작업마다 branch를 만든다.
- branch 이름: `담당/이슈번호-작업내용` (예: `go/7-ecos-sample`, `docs/5-claude-md`)
- commit, push, PR, merge는 사람이 한다.
  - 사람이 요청한 경우에만 commit, push, PR 생성까지 할 수 있다. merge는 하지 않는다.
- 작업이 끝나면 commit·push·PR 명령과 메시지 초안을 안내한다. PR 설명에는 `Closes #이슈번호`를 넣는다.
- PR 자동 검사가 실패(❌)하면 merge를 안내하지 않는다.

## 작업 결과 보고

```markdown
## 작업 결과

- 변경 내용:
- 변경 파일:
- 테스트:
- 남은 문제:
```
