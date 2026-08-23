// SPDX-License-Identifier: MIT

#[test_only]
module grepquest::social_tests;

use grepquest::social::{Self, AdminCap, PlayerProfile, Season, VerifierCap};
use sui::test_scenario as ts;

const ADMIN: address = @0xAD;
const PLAYER: address = @0xA;

fun digest(value: u8): vector<u8> {
    vector[
        value, value, value, value, value, value, value, value,
        value, value, value, value, value, value, value, value,
        value, value, value, value, value, value, value, value,
        value, value, value, value, value, value, value, value,
    ]
}

#[test]
fun profile_retains_only_the_best_season_score() {
    let mut scenario = ts::begin(ADMIN);
    social::init_for_testing(scenario.ctx());

    scenario.next_tx(ADMIN);
    {
        let cap: AdminCap = scenario.take_from_sender();
        social::create_season(&cap, b"core-v1", digest(1), digest(2), 0, 10, scenario.ctx());
        social::create_verifier_cap(&cap, ADMIN, scenario.ctx());
        scenario.return_to_sender(cap);
    };

    scenario.next_tx(PLAYER);
    social::create_profile(scenario.ctx());

    scenario.next_tx(PLAYER);
    {
        let mut profile: PlayerProfile = scenario.take_from_sender();
        let season: Season = scenario.take_immutable();
        social::submit_score(
            &mut profile, &season, b"core-v1", digest(1), digest(2), vector[1], 4, 900, digest(3), 0,
            90, 80, 7, 1200, scenario.ctx(),
        );
        social::submit_score(
            &mut profile, &season, b"core-v1", digest(1), digest(2), vector[0], 3, 700, digest(4), 0,
            70, 60, 4, 1000, scenario.ctx(),
        );
        assert!(social::profile_owner(&profile) == PLAYER);
        assert!(social::best_season_score(&profile) == 900);
        assert!(social::latest_receipt_sha256(&profile) == digest(3));
        scenario.return_to_sender(profile);
        ts::return_immutable(season);
    };

    scenario.next_tx(ADMIN);
    {
        let verifier: VerifierCap = scenario.take_from_sender();
        let season: Season = scenario.take_immutable();
        social::submit_verified_score(
            &verifier, PLAYER, &season, b"core-v1", digest(1), digest(2), 4, 900, digest(3), 1,
            90, 80, 7, 1200, scenario.ctx(),
        );
        scenario.return_to_sender(verifier);
        ts::return_immutable(season);
    };

    scenario.end();
}

#[test]
#[expected_failure(abort_code = 5)]
fun player_cannot_claim_verified_tier() {
    let mut scenario = ts::begin(ADMIN);
    social::init_for_testing(scenario.ctx());

    scenario.next_tx(ADMIN);
    {
        let cap: AdminCap = scenario.take_from_sender();
        social::create_season(&cap, b"core-v1", digest(1), digest(2), 0, 10, scenario.ctx());
        scenario.return_to_sender(cap);
    };

    scenario.next_tx(PLAYER);
    social::create_profile(scenario.ctx());
    scenario.next_tx(PLAYER);
    let mut profile: PlayerProfile = scenario.take_from_sender();
    let season: Season = scenario.take_immutable();
    social::submit_score(
        &mut profile, &season, b"core-v1", digest(1), digest(2), vector[1], 4, 900, digest(3), 1,
        90, 80, 7, 1200, scenario.ctx(),
    );
    abort 999
}
