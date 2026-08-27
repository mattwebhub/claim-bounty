# WEB-STATE-001 — Single state ownership

TanStack Query owns server resources, the router owns shareable navigation state, Zustand owns synchronous workspace workflow state, React Hook Form owns form drafts, and component state owns private presentation details.

Do not mirror query resources into Zustand. A controller may synchronize an editable draft with a query result only through explicit load, dirty, save, conflict, and reset transitions covered by behavior tests.
