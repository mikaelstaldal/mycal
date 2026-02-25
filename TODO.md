# TODO

## Current

- [x] Add a progress spinner when bulk importing events
- [x] Display past events with a lighter color
- [x] Add support for iCalendar VTIMEZONE definitions for import and export

## iCalendar (RFC 5545) compliance

### Recurrence
- [x] Support RRULE:INTERVAL=N, recurring events every N day/week/month/year
- [x] Support RRULE BYDAY, BYMONTH, BYMONTHDAY and other BY* parameters for complex patterns
- [x] Support EXDATE, removing instances of recurring events
- [x] Support RDATE for explicit additional recurrence dates
- [ ] Support RECURRENCE-ID for identifying and editing individual instances of recurring events

### Event properties
- [ ] Support STATUS property (TENTATIVE, CONFIRMED, CANCELLED)
- [ ] Support CLASS property (PUBLIC, PRIVATE, CONFIDENTIAL)
- [ ] Support TRANSP property (OPAQUE, TRANSPARENT)
- [ ] Support SEQUENCE property for revision tracking
- [ ] Support CATEGORIES property for event tagging
- [ ] Support URL property for reference links
- [ ] Support ATTACH property for file attachments or URLs
- [ ] Support PRIORITY property (0-9)
- [ ] Support DURATION as alternative to DTEND
- [ ] Support RELATED-TO property for parent/child event relationships

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

## Later

- [ ] Native Android app
