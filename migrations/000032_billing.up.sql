-- CampusDesk's own SaaS subscription billing (schools paying to use the
-- app), separate from the existing fee module (schools collecting tuition
-- fees from parents). Plans are configured here and linked to a matching
-- plan created on Razorpay's own dashboard via razorpay_plan_id; actual
-- subscriptions are created through the Razorpay API at runtime and mirrored
-- here so the app has its own record independent of calling Razorpay again.

CREATE TABLE subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    -- Smallest currency unit (paise), matching how Razorpay itself takes
    -- amounts -- avoids float rounding on money.
    price_monthly_paise INTEGER NOT NULL DEFAULT 0,
    price_annual_paise INTEGER NOT NULL DEFAULT 0,
    max_students INTEGER, -- NULL = unlimited
    features JSONB NOT NULL DEFAULT '[]',
    razorpay_plan_id_monthly VARCHAR(64),
    razorpay_plan_id_annual VARCHAR(64),
    is_active BOOLEAN NOT NULL DEFAULT true,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    billing_cycle VARCHAR(10) NOT NULL DEFAULT 'monthly', -- 'monthly' | 'annual'
    -- Mirrors Razorpay's own subscription status vocabulary directly
    -- (created, authenticated, active, pending, halted, cancelled,
    -- completed, expired) rather than inventing a separate one, so the
    -- webhook handler can just copy the field across.
    status VARCHAR(20) NOT NULL DEFAULT 'created',
    razorpay_subscription_id VARCHAR(64) UNIQUE,
    razorpay_customer_id VARCHAR(64),
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    cancel_at_cycle_end BOOLEAN NOT NULL DEFAULT false,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One school has at most one subscription row at a time; upgrading/
-- downgrading updates this row rather than creating a new one (a fresh
-- Razorpay subscription still gets created for the new plan, but the old
-- one's id is overwritten here once the new one activates).
CREATE UNIQUE INDEX idx_subscriptions_school ON subscriptions(school_id);

CREATE TABLE subscription_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    razorpay_payment_id VARCHAR(64) UNIQUE,
    razorpay_invoice_id VARCHAR(64),
    amount_paise INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL, -- 'captured' | 'failed' | 'refunded'
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscription_payments_subscription ON subscription_payments(subscription_id);
