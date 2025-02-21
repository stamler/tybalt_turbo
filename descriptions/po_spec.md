# Purchase Order Specification

## PO types and their behaviours

| PO Type    | Description                                                                                   | May be Closed Manually                               | May be Canceled if status is Active | Closed automatically                    | Can be converted to different PO type             | Approval tier required                 |
|------------|-----------------------------------------------------------------------------------------------|------------------------------------------------------|-------------------------------------|-----------------------------------------|---------------------------------------------------|----------------------------------------|
| Normal     | Valid for a single expense                                                                    | No                                                   | Yes, if no expenses are associated  | When an expense is committed against it | Yes, to Cumulative by payables_admin claim holder | Based on PO amount                     |
| Recurring  | Valid for a fixed number of expenses not exceeding specified value                            | Yes, if status is Active and > 0 expenses associated | Yes, if no expenses are associated  | When the final expense is committed     | No                                                | Based on PO amount * number of periods |
| Cumulative | Valid for an unlimited number of expenses where sum of values does not exceed specified value | Yes, if status is Active and > 0 expenses associated | Yes, if no expenses are associated  | When the maximum amount is reached      | No                                                | Based on PO amount                     |

## Approval Tiers

| Tier | Description                                   |
|------|-----------------------------------------------|
| 1    | PO amount < TIER_1_PO_LIMIT                   |
| 2    | TIER_1_PO_LIMIT ≤ PO amount < TIER_2_PO_LIMIT |
| 3    | PO amount ≥ TIER_1_PO_LIMIT                   |

*tiers are adjustable through a software update via constants rather
than stored in the database and currently have values of 500 and 2500*
