-- Peer2Paper user-owned audit intake records.
-- Run with `supabase db push` or paste into the Supabase SQL editor.

create extension if not exists "pgcrypto";

create type public.audit_status as enum (
  'submitted',
  'in_review',
  'running',
  'completed',
  'needs_input'
);

create table public.audit_requests (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references auth.users(id) on delete cascade,
  title text not null check (char_length(title) between 3 and 160),
  claim text not null check (char_length(claim) between 20 and 5000),
  paper_url text check (paper_url is null or paper_url ~ '^https?://'),
  materials_url text check (materials_url is null or materials_url ~ '^https?://'),
  notes text check (notes is null or char_length(notes) <= 5000),
  status public.audit_status not null default 'submitted',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index audit_requests_user_created_idx
  on public.audit_requests (user_id, created_at desc);

alter table public.audit_requests enable row level security;

revoke all on table public.audit_requests from anon, authenticated;
grant select, insert on table public.audit_requests to authenticated;

create policy "Users can view their audit requests"
  on public.audit_requests for select
  to authenticated
  using ((select auth.uid()) = user_id);

create policy "Users can create their own audit requests"
  on public.audit_requests for insert
  to authenticated
  with check ((select auth.uid()) = user_id and status = 'submitted');

-- Status and request edits are intentionally service-managed. Do not add a
-- broad client update policy; expose explicit reviewed operations instead.

create or replace function public.set_updated_at()
returns trigger
language plpgsql
security invoker
set search_path = ''
as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

create trigger audit_requests_updated_at
before update on public.audit_requests
for each row execute function public.set_updated_at();
