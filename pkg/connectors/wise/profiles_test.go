package wise

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/pkg/connectors/wise/client"
	"github.com/formancehq/payments/pkg/connector"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Wise Plugin Profiles", func() {
	var (
		ctrl *gomock.Controller
		plg  connector.Plugin
		m    *client.MockClient
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetch next profiles", func() {
		var (
			profiles []client.Profile
		)

		BeforeEach(func() {
			profiles = []client.Profile{
				{ID: 14556, Type: "type1"},
				{ID: 3334, Type: "type2"},
			}
		})

		It("replies with unimplemented when unknown other type in request", func(ctx SpecContext) {
			req := connector.FetchNextOthersRequest{
				State:    json.RawMessage(`{}`),
				PageSize: len(profiles),
			}
			_, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})

		It("fetches profiles from wise", func(ctx SpecContext) {
			req := connector.FetchNextOthersRequest{
				State:    json.RawMessage(`{}`),
				Name:     "fetch_profiles",
				PageSize: len(profiles),
			}
			m.EXPECT().GetProfiles(gomock.Any()).Return(
				profiles,
				nil,
			)

			res, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(BeTrue())
			Expect(res.Others).To(HaveLen(req.PageSize))
			Expect(res.Others[0].ID).To(Equal(fmt.Sprint(profiles[0].ID)))
			Expect(res.Others[1].ID).To(Equal(fmt.Sprint(profiles[1].ID)))

			var state profilesState

			err = json.Unmarshal(res.NewState, &state)
			Expect(err).To(BeNil())
			Expect(fmt.Sprint(state.LastProfileID)).To(Equal(res.Others[len(res.Others)-1].ID))
		})
	})
})
