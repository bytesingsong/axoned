package predicate

import (
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/store/metrics"

	"github.com/ichiban/prolog"
	"github.com/ichiban/prolog/engine"

	. "github.com/smartystreets/goconvey/convey"

	"cosmossdk.io/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/store"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/okp4/okp4d/x/logic/testutil"
	"github.com/okp4/okp4d/x/logic/types"
)

func TestHexBytesPredicate(t *testing.T) {
	Convey("Given a test cases", t, func() {
		cases := []struct {
			program     string
			query       string
			wantResult  []types.TermResults
			wantError   error
			wantSuccess bool
		}{
			{
				query: `hex_bytes(Hex,
				[44,38,180,107,104,255,198,143,249,155,69,60,29,48,65,52,19,66,45,112,100,131,191,160,249,138,94,136,98,102,231,174]).`,
				wantResult:  []types.TermResults{{"Hex": "'2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae'"}},
				wantSuccess: true,
			},
			{
				query: `hex_bytes('2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae', Bytes).`,
				wantResult: []types.TermResults{{
					"Bytes": "[44,38,180,107,104,255,198,143,249,155,69,60,29,48,65,52,19,66,45,112,100,131,191,160,249,138,94,136,98,102,231,174]",
				}},
				wantSuccess: true,
			},
			{
				query: `hex_bytes('2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae',
				[44,38,180,107,104,255,198,143,249,155,69,60,29,48,65,52,19,66,45,112,100,131,191,160,249,138,94,136,98,102,231,174]).`,
				wantResult:  []types.TermResults{{}},
				wantSuccess: true,
			},
			{
				query: `hex_bytes('3c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae',
				[44,38,180,107,104,255,198,143,249,155,69,60,29,48,65,52,19,66,45,112,100,131,191,160,249,138,94,136,98,102,231,174]).`,
				wantSuccess: false,
			},
			{
				query: `hex_bytes('fail',
				[44,38,180,107,104,255,198,143,249,155,69,60,29,48,65,52,19,66,45,112,100,131,191,160,249,138,94,136,98,102,231,174]).`,
				wantError: fmt.Errorf("error(domain_error(encoding(hex),fail),[%s],hex_bytes/2)",
					strings.Join(strings.Split("encoding/hex: invalid byte: U+0069 'i'", ""), ",")),
				wantSuccess: false,
			},
			{
				query: `hex_bytes('2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae',
				[45,38,180,107,104,255,198,143,249,155,69,60,29,48,65,52,19,66,45,112,100,131,191,160,249,138,94,136,98,102,231,174]).`,
				wantSuccess: false,
			},
			{
				query: `hex_bytes('2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae',
				[345,38,'hey',107,104,255,198,143,249,155,69,60,29,48,65,52,19,66,45,112,100,131,191,160,249,138,94,136,98,102,231,174]).`,
				wantSuccess: false,
				wantError:   fmt.Errorf("error(type_error(byte,345),hex_bytes/2)"),
			},
		}
		for nc, tc := range cases {
			Convey(fmt.Sprintf("Given the query #%d: %s", nc, tc.query), func() {
				Convey("and a context", func() {
					db := dbm.NewMemDB()
					stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
					ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

					Convey("and a vm", func() {
						interpreter := testutil.NewLightInterpreterMust(ctx)
						interpreter.Register2(engine.NewAtom("hex_bytes"), HexBytes)

						err := interpreter.Compile(ctx, tc.program)
						So(err, ShouldEqual, nil)

						Convey("When the predicate is called", func() {
							sols, err := interpreter.QueryContext(ctx, tc.query)

							Convey("Then the error should be nil", func() {
								So(err, ShouldEqual, nil)
								So(sols, ShouldNotBeNil)

								Convey("and the bindings should be as expected", func() {
									checkSolutions(sols, tc.wantResult, tc.wantSuccess, tc.wantError)
								})
							})
						})
					})
				})
			})
		}
	})
}

func checkSolutions(sols *prolog.Solutions, wantResult []types.TermResults, wantSuccess bool, wantError error) {
	var got []types.TermResults
	for sols.Next() {
		m := types.TermResults{}
		err := sols.Scan(m)
		So(err, ShouldEqual, nil)

		got = append(got, m)
	}
	if wantError != nil {
		So(sols.Err(), ShouldNotEqual, nil)
		So(sols.Err().Error(), ShouldEqual, wantError.Error())
	} else {
		So(sols.Err(), ShouldEqual, nil)

		if wantSuccess {
			So(len(got), ShouldEqual, len(wantResult))
			for iGot, resultGot := range got {
				for varGot, termGot := range resultGot {
					So(testutil.ReindexUnknownVariables(termGot), ShouldEqual, wantResult[iGot][varGot])
				}
			}
		} else {
			So(len(got), ShouldEqual, 0)
		}
	}
}
