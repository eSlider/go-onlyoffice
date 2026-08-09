# CRM associations (company ↔ person ↔ deal ↔ project ↔ invoice ↔ mail)

Operational rules for the `oo` CLI and this library. Business SSOT remains
OnlyOffice Workspace CRM + Projects.

## Canonical graph

One **legal company** owns the relationship. Do not invent a second “bill-to”
company just for PDF layout.

```text
Company
├── Person (buyer contact)          oo persons create --company-id
├── Opportunity / Deal              oo opportunities … ; member-add company + person
├── Project (hub)                   oo projects … ; contacts add company + person
│     └── Epic + subtasks
└── Invoice (Draft → …)             oo invoices create --contact COMPANY --opportunity DEAL
      └── PDF file                  oo invoices pdf ID
            └── Mail draft          oo mails draft-invoice --invoice ID --to …
```

| Layer | CLI | Must link |
|-------|-----|-----------|
| Company | `oo companies create` | website, email, phone, **one** Billing address |
| Person | `oo persons create --company-id` / `oo persons update ID` | job title; never encode employer in `lastName`; **update uses JSON** (form PUT ignores `companyId`/`about`) |
| Deal | `oo opportunities create` + `member-add` | company **and** person as members |
| Project | `oo projects create` + `contacts add` | same company + person |
| Invoice | `oo invoices create --contact COMPANY --opportunity DEAL` | `entityId` at **create** |
| Mail | `oo mails draft-invoice` | attach current PDF; **do not send** until confirmed |

UI checks (same company card):

- `#contacts` → person  
- `#deals` → opportunity  
- `#projects` → hub project  
- `#invoices` on the **deal** → invoice (needs `entity`)  
- `#files` → preferably **one** current invoice PDF

**Project Team ≠ Project Contacts.** Team = portal users. CRM people/companies
show under the project **Contacts** tab (`oo projects contacts list`).

## Hard rules

1. **One company per legal entity.** Duplicate “bill-to” contacts empty Deals /
   Projects / Contacts tabs and break merge. Prefer
   `oo contacts merge FROM INTO` (keeps `INTO`) or `oo companies dedupe`.
2. **Link invoice → deal at create.**  
   `POST /crm/invoice` with `entityId` + `entityType: 0` (Opportunity).  
   `oo invoices update … --opportunity` often returns **400**
   (“Value does not fall within the expected range”). If the link is missing,
   delete the Draft and recreate with `--opportunity`.
3. **Bill To = company id**, not a throwaway contact. Person stays under the
   company (`companyId`). Optional `consigneeId` for Empfänger when the portal
   template prints it.
4. **Stay Draft until mail is ready.** Billed (`status id=2`) is **not editable**
   via content PUT. Going Billed → Draft via `…/crm/invoice/status/1` usually
   **does not work** — delete + recreate Draft instead.
5. **Do not regenerate PDF in a loop** without cleanup. Each
   `GET …/crm/invoice/{id}/pdf` attaches a new file to the company (and often
   the deal). Keep `invoice.fileID`; delete older PDFs with
   `oo invoices pdf-cleanup ID` / Documents `fileops/delete`.

## Invoice PDF quirks

| Symptom | Workaround |
|---------|------------|
| Cached / stale PDF | Touch invoice (Draft PUT that clears `fileID`), then `GET …/pdf` — `oo invoices pdf ID --force` |
| Billing address missing on **new** PDFs | Temporary multiline `companyName` (`Line1\nLine2\n…`) on the **canonical** company → force PDF → restore clean name. Cached `fileID` keeps the multiline Bill To. |
| Separate bill-to company for newlines | **Forbidden** — merge back to the real company |
| Invoice **number** won’t change on PUT | Delete Draft and recreate with the desired number |
| Notizen / Bedingungen spacing | Leading `\n` and blank lines only — no HTML (tags print literally) |
| Issuer street lines | Organisation profile address (`street` with `\n`), not only terms |

Status ids commonly used: `1` Draft, `2` Billed, `3` Rejected, `4` Paid.

## Mail quirks

| Symptom | Workaround |
|---------|------------|
| Signature / body doubles chat URL | Put chat in **one** place only. UI drafts: signature. API send: body (API **does not** append signature). |
| Signature / body cuts URL at `#` | Plain text URLs — avoid `<a href="…#…">` (or encode `#` as `%23` in href) |
| German letter spacing | Blank `<p>&nbsp;</p>` between blocks (`MailHTMLWithBlankParagraphs`) |
| Send | `PUT /api/2.0/mail/messages/send.json` with `id/from/to/subject/body`; omit empty `cc`/`bcc`. Never auto-send; draft only until the human confirms |

Prefer OnlyOffice Mail (`/addons/mail/#drafts`) for invoice delivery until confirmed.

## Project / task quirks

- Hub title: `CC | Company` (e.g. `DE | Acme GmbH`).
- Streams = epics/tasks under the hub, not a third title segment (unless the
  project itself is a named delivery stream).
- Closing a **subtask**:  
  `PUT /api/2.0/project/task/{epicId}/{subtaskId}/status` with `status=2`.  
  `oo tasks update SUBTASK -s closed` returns **404** for subtasks.
- After deleting a CRM contact, `GET /project/contact/{deletedId}` may still
  return projects (ghost). Official project contact list should only show live
  ids; unlink may 400 if the contact is gone.

## Merge / cleanup cheat sheet

```bash
# Keep the preferred company (INTO), drop the duplicate (FROM)
oo contacts merge FROM_ID INTO_ID

# Or by normalized name (careful — whole CRM)
oo companies dedupe

# Invoice ↔ deal must exist at create
oo invoices create --number P-YYYY-NN --contact COMPANY_ID --item ITEM_ID \
  --price 300 --opportunity DEAL_ID --language de-DE …

# Fresh PDF + prune older PDFs on company/deal
oo invoices pdf INVOICE_ID --force
oo invoices pdf-cleanup INVOICE_ID

# Mail draft (no send)
oo mails draft-invoice --invoice INVOICE_ID --to billing@example.com
```

## Related

- README § invoices / mail / CRM cleanup  
- Personal workspace tooling (disk inventory, dossier sync): private
  `git.produktor.io/eSlider/oo-workspace` (`oow` CLI)
