# TODO

## Current

- [x] Add a command line option to set HTTP basic auth realm
- [x] Validate that a single event import is not modifying an existing event (e.g. RECURRENCE-ID)
- [x] Add a command line option to export all events to an .ics file
- [x] Sometimes you get a meeting invitation as an email with an .ics file attached or an iCalendar link. Have a way to quickly import that as a single event.

## iCalendar (RFC 5545) compliance

### Recurrence
- [x] Support RRULE:INTERVAL=N, recurring events every N day/week/month/year
- [x] Support RRULE BYDAY, BYMONTH, BYMONTHDAY and other BY* parameters for complex patterns
- [x] Support EXDATE, removing instances of recurring events
- [x] Support RDATE for explicit additional recurrence dates
- [x] Support RECURRENCE-ID for identifying and editing individual instances of recurring events

### Event properties
- [ ] Support STATUS property (TENTATIVE, CONFIRMED, CANCELLED)
- [ ] Support CLASS property (PUBLIC, PRIVATE, CONFIDENTIAL)
- [ ] Support TRANSP property (OPAQUE, TRANSPARENT)
- [ ] Support SEQUENCE property for revision tracking
- [x] Support CATEGORIES property for event tagging
- [x] Support URL property for reference links
- [ ] Support ATTACH property for file attachments or URLs
- [ ] Support PRIORITY property (0-9)
- [x] Support DURATION as alternative to DTEND
- [ ] Support RELATED-TO property for parent/child event relationships
- [x] Support COLOR according to RFC 7986

### Multi-user
- [ ] Support ORGANIZER property
- [ ] Support ATTENDEE property with PARTSTAT, RSVP, ROLE
- [ ] Support METHOD values beyond PUBLISH (REQUEST, REPLY, CANCEL)

### Alarms
- [ ] Support AUDIO and EMAIL alarm actions (currently only DISPLAY)

### Other
- [ ] Support VFREEBUSY component for availability scheduling
- [ ] Support CONTACT property
- [ ] Support RESOURCES property
- [ ] Support REFRESH-INTERVAL for subscription feed optimization
- [x] Support VTIMEZONE definitions for import and export

## Later

- [ ] Native Android app
